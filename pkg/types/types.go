package types

import "google.golang.org/protobuf/reflect/protoreflect"

type Transaction struct {
	Nonce           string `json:"nonce"`
	Raw             string `json:"raw"`
	Code            uint32 `json:"code"`
	TxID            string `json:"tx_id"`
	Tx              any    `json:"tx"`
	TxResult        any    `json:"tx_result"`
	Metadata        any    `json:"metadata"`
	ShortTermOrders any    `json:"short_term_orders"`
}

type Block struct {
	Nonce string `json:"nonce"`
	Block any    `json:"block"`
}

type Mempool struct {
	Transactions []*Transaction `json:"txs"`
}

func (*Mempool) ProtoReflect() protoreflect.Message { return nil }
