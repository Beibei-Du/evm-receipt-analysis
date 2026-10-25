// receiptflows reports observed ERC-20 transfers; it does not classify strategy profit.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"sort"
	"strings"
)

const (
	transferTopic    = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	addressHexLength = 40
	wordHexLength    = 64
)

type logEntry struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}
type receipt struct {
	TransactionHash   string     `json:"transactionHash"`
	From              string     `json:"from"`
	To                string     `json:"to"`
	Status            string     `json:"status"`
	GasUsed           string     `json:"gasUsed"`
	EffectiveGasPrice string     `json:"effectiveGasPrice"`
	Logs              []logEntry `json:"logs"`
}
type flow struct {
	Address string `json:"address"`
	Token   string `json:"token"`
	NetRaw  string `json:"net_raw"`
}

func integer(s string) (*big.Int, error) {
	if !validHex(s) {
		return nil, fmt.Errorf("invalid hex quantity")
	}
	n, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
	if !strings.HasPrefix(s, "0x") || !ok {
		return nil, fmt.Errorf("invalid hex quantity")
	}
	return n, nil
}

func validHex(s string) bool {
	if !strings.HasPrefix(s, "0x") || len(s) < 3 {
		return false
	}
	for _, ch := range s[2:] {
		if !(ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f' || ch >= 'A' && ch <= 'F') {
			return false
		}
	}
	return true
}

func validAddressTopic(s string) bool {
	return len(s) == wordHexLength+2 && validHex(s) && s[2:len(s)-addressHexLength] == strings.Repeat("0", wordHexLength-addressHexLength)
}
func flows(r receipt) ([]flow, error) {
	totals := map[string]*big.Int{}
	for _, l := range r.Logs {
		if len(l.Topics) == 0 || strings.ToLower(l.Topics[0]) != transferTopic {
			continue
		}
		// ERC-721 Transfer has four topics; it is not an ERC-20 cashflow.
		if len(l.Topics) == 4 {
			continue
		}
		if len(l.Topics) != 3 || len(l.Data) != wordHexLength+2 || !validAddressTopic(l.Topics[1]) || !validAddressTopic(l.Topics[2]) || len(l.Address) != addressHexLength+2 || !validHex(l.Address) {
			return nil, fmt.Errorf("malformed ERC-20 Transfer")
		}
		n, err := integer(l.Data)
		if err != nil {
			return nil, err
		}
		for i, sign := range []int64{-1, 1} {
			key := strings.ToLower("0x" + l.Topics[i+1][len(l.Topics[i+1])-addressHexLength:] + "/" + l.Address)
			if totals[key] == nil {
				totals[key] = new(big.Int)
			}
			totals[key].Add(totals[key], new(big.Int).Mul(n, big.NewInt(sign)))
		}
	}
	keys := make([]string, 0, len(totals))
	for k := range totals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []flow{}
	for _, k := range keys {
		if totals[k].Sign() != 0 {
			a := strings.Split(k, "/")
			out = append(out, flow{a[0], a[1], totals[k].String()})
		}
	}
	return out, nil
}
func run(input io.Reader, output io.Writer) error {
	var envelope struct {
		Result receipt `json:"result"`
	}
	decoder := json.NewDecoder(input)
	err := decoder.Decode(&envelope)
	if err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("expected one receipt envelope")
	}
	r := envelope.Result
	if r.TransactionHash == "" || r.Status != "0x1" {
		return fmt.Errorf("successful receipt envelope required")
	}
	f, err := flows(r)
	if err != nil {
		return err
	}
	gas, err := integer(r.GasUsed)
	if err != nil {
		return err
	}
	price, err := integer(r.EffectiveGasPrice)
	if err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(struct {
		Tx                   string `json:"transaction_hash"`
		Sender               string `json:"sender"`
		Executor             string `json:"executor"`
		ReceiptGasProductWei string `json:"receipt_gas_product_wei"`
		Flows                []flow `json:"erc20_net_transfers"`
		Limitation           string `json:"limitation"`
	}{r.TransactionHash, r.From, r.To, new(big.Int).Mul(gas, price).String(), f, "Transfers are not independently checked balance deltas; native/internal transfers, token semantics, fee components and ownership need separate validation. No USD valuation or net-profit claim."})
}
func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: receiptflows receipt.json (a JSON-RPC receipt envelope)")
		os.Exit(1)
	}
	input, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer input.Close()
	if err := run(input, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
