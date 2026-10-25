# EVM receipt analysis

A small offline Go CLI for inspecting ERC-20 transfers in a transaction receipt. Extracted from my personal Robinhood Chain arbitrage research tooling.

The tool nets observed transfers by address and token, reports the receipt gas arithmetic, and produces deterministic JSON. It deliberately does not turn transfer logs into a strategy-profit claim.

## Run

Requires Go 1.26 or later. Uses only the Go standard library. No RPC endpoint, credentials or wallet are needed.

```bash
go run . examples/receipt.json
```

Or build the CLI:

```bash
go build -o receiptflows .
./receiptflows examples/receipt.json
```

Input is one JSON-RPC receipt envelope (`{"result": {...}}`) for a successful transaction. Normal additional receipt fields are accepted. Failed receipts and malformed transfer events are rejected.

The included receipt is synthetic. It contains a 1,000-unit transfer, a 900-unit repayment and an NFT transfer. The two ERC-20 address flows net to -100 and +100; the NFT is excluded.

## Output

| Field | Meaning |
|---|---|
| `transaction_hash` | Receipt transaction identifier |
| `sender`, `executor` | Receipt `from` and `to`, not an ownership conclusion |
| `receipt_gas_product_wei` | Exact `gasUsed × effectiveGasPrice` |
| `erc20_net_transfers` | Signed raw-unit transfer totals by address and token |
| `limitation` | The boundary between observed logs and economic conclusions |

Amounts are decimal strings computed using `math/big`, with no floating-point conversion. Token decimals and prices are not guessed. Rows are sorted by address and token so repeated runs have stable output.

## What the receipt proves

ERC-20 logs show transfers emitted by token contracts. They are not independently verified balance deltas. Native/internal transfers, token-specific behavior, asset ownership and additional chain fee components require separate evidence. The gas product is not necessarily the full transaction fee on every chain. There is no USD valuation or net-profit calculation.

The tool is useful as an inspection step in replay and transaction-outcome analysis. It does not replay EVM execution, discover opportunities, bid or send transactions.

## Tests

```bash
go test ./...
go vet ./...
```

Tests cover repayments, NFT exclusion, exact gas products, 256-bit quantities, malformed topics and unsuccessful or incomplete receipt envelopes.

## Origin

The transfer-netting algorithm and original repayment/NFT tests were extracted from my private arbitrage research command. This standalone version adds explicit input validation, synthetic examples and tests. No trading decisions, route-selection logic, execution contracts or deployment configuration are included.

Built by [Adam Du](https://github.com/Beibei-Du). MIT licensed; see [LICENSE](LICENSE).
