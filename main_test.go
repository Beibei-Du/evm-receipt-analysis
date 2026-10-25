package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestTransfersNetRepaymentAndExcludeNFT(t *testing.T) {
	a := "0x" + strings.Repeat("0", 63) + "1"
	b := "0x" + strings.Repeat("0", 63) + "2"
	token := "0x" + strings.Repeat("3", 40)
	log := func(n string, reverse bool) logEntry {
		topics := []string{transferTopic, a, b}
		if reverse {
			topics = []string{transferTopic, b, a}
		}
		return logEntry{token, topics, "0x" + strings.Repeat("0", 64-len(n)) + n}
	}
	r := receipt{Logs: []logEntry{log("64", false), log("5a", true), {token, []string{transferTopic, a, b, a}, "0x"}}}
	f, err := flows(r)
	if err != nil || len(f) != 2 || f[0].NetRaw != "-10" || f[1].NetRaw != "10" {
		t.Fatalf("%+v %v", f, err)
	}
}
func TestMalformedTransferRejected(t *testing.T) {
	_, err := flows(receipt{Logs: []logEntry{{Topics: []string{transferTopic}, Data: "0x"}}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSampleReceiptExactGasAndFlows(t *testing.T) {
	in, err := os.Open("examples/receipt.json")
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	var out bytes.Buffer
	if err := run(in, &out); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Gas   string `json:"receipt_gas_product_wei"`
		Flows []flow `json:"erc20_net_transfers"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Gas != "21000000000000" {
		t.Fatalf("incorrect gas product: %s", report.Gas)
	}
	if len(report.Flows) != 2 || report.Flows[0].NetRaw != "-100" || report.Flows[1].NetRaw != "100" {
		t.Fatalf("unexpected flows: %+v", report.Flows)
	}
}

func TestRejectMalformedQuantities(t *testing.T) {
	for _, value := range []string{"", "0x", "12", "0x-1", "0x+1", "0x1g"} {
		if _, err := integer(value); err == nil {
			t.Errorf("accepted malformed quantity %q", value)
		}
	}
	const large = "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	n, err := integer(large)
	if err != nil || n.BitLen() != 256 {
		t.Fatalf("large integer lost precision: %v %v", n, err)
	}
}

func TestRejectMalformedAddressTopics(t *testing.T) {
	a := "0x" + strings.Repeat("0", 63) + "1"
	b := "0x" + strings.Repeat("0", 63) + "2"
	token := "0x" + strings.Repeat("3", 40)
	for _, invalid := range []string{"0x" + strings.Repeat("g", 64), "0x1" + strings.Repeat("0", 63)} {
		_, err := flows(receipt{Logs: []logEntry{{Address: token, Topics: []string{transferTopic, invalid, b}, Data: a}}})
		if err == nil {
			t.Errorf("accepted malformed topic: %s", invalid)
		}
	}
}

func TestReceiptEnvelopeRequired(t *testing.T) {
	for _, input := range []string{`{}`, `{"result":{"status":"0x0","transactionHash":"sample"}}`, `{} {}`} {
		var out bytes.Buffer
		if err := run(strings.NewReader(input), &out); err == nil {
			t.Errorf("accepted invalid envelope: %s", input)
		}
		if out.Len() != 0 {
			t.Fatal("emitted a report for an invalid receipt")
		}
	}
}
