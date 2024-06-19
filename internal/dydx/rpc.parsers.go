package dydx

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	IBCTypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	"gitlab.com/syntropynet/amberdm/publisher/dydx-publisher/pkg/types"
	"log"
	"strings"

	tmtypes "github.com/cometbft/cometbft/types"
	cosmotypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	clobtypes "github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

type TxProtoGetter interface {
	GetProtoTx() *tx.Tx
}

type IBCDenomTrace map[string]IBCTypes.DenomTrace

func (c IBCDenomTrace) Add(denom string) {
	if !strings.HasPrefix(strings.ToLower(denom), "ibc/") {
		return
	}
	c[denom] = IBCTypes.DenomTrace{}
}

func (c *rpc) decodeTransaction(txRaw []byte) (cosmotypes.Tx, error) {
	decoder := c.enccfg.TxConfig.TxDecoder()
	return decoder(txRaw)
}

func decodeOperationRawShortTermOrderPlacementBytes(
	bytes []byte,
	decoder sdk.TxDecoder,
) (*clobtypes.MsgPlaceOrder, error) {
	txdecoder, err := decoder(bytes)
	if err != nil {
		return nil, err
	}

	msgs := txdecoder.GetMsgs()
	if len(msgs) != 1 {
		return nil, fmt.Errorf("expected 1 msg, got %d", len(msgs))
	}

	msg, ok := msgs[0].(*clobtypes.MsgPlaceOrder)
	if !ok {
		return nil, fmt.Errorf("expected MsgPlaceOrder, got %T", msgs[0])
	}

	return msg, nil
}

func (c *rpc) getDenomsFromTransactions(tx cosmotypes.Tx) (map[string]IBCTypes.DenomTrace, error) {
	ibcTrace := make(IBCDenomTrace)
	for _, msg := range tx.GetMsgs() {
		switch m := msg.(type) {
		case *banktypes.MsgMultiSend:
			for _, input := range m.Inputs {
				ibcTrace.Add(input.Coins[0].Denom)
			}
			for _, output := range m.Outputs {
				ibcTrace.Add(output.Coins[0].Denom)
			}
		case *banktypes.MsgSend:
			ibcTrace.Add(m.Amount[0].Denom)
		case *ibctransfertypes.MsgTransfer:
			ibcTrace.Add(m.Token.Denom)
		}
	}

	for denom := range ibcTrace {
		res, err := c.getDenomTraceFromCache(denom)
		if err != nil {
			log.Printf("getDenomTraceFromCache failed for denom %s: \n %s", denom, err.Error())
		} else {
			ibcTrace[denom] = res
			log.Println("DenomTrace: ", denom, res.String())
		}
	}

	return ibcTrace, nil
}

func (c *rpc) translateTransaction(
	txRaw []byte, txid, nonce string, txResult *abci.TxResult, code *uint32,
) *types.Transaction {
	transaction := &types.Transaction{
		Nonce:    nonce,
		TxID:     txid,
		Raw:      hex.EncodeToString(txRaw),
		TxResult: txResult,
	}
	if code != nil {
		transaction.Code = *code
	}

	decodedTx, err := c.decodeTransaction(txRaw)
	if err != nil {
		log.Println("Decode Transaction failed:", err.Error())
		return transaction
	}

	shortTermOrders := []clobtypes.MsgPlaceOrder{}

	for _, msg := range decodedTx.GetMsgs() {
		if placeOrderMsg, ok := msg.(*clobtypes.MsgProposedOperations); ok {
			for _, op := range placeOrderMsg.OperationsQueue {
				shortTermOrderData := op.GetShortTermOrderPlacement()
				if len(shortTermOrderData) > 0 {
					decodedShortTermOrder, err := decodeOperationRawShortTermOrderPlacementBytes(shortTermOrderData, c.enccfg.TxConfig.TxDecoder())
					if err == nil {
						shortTermOrders = append(shortTermOrders, *decodedShortTermOrder)
					}
				}
			}
		}
	}

	transaction.ShortTermOrders = shortTermOrders

	ibcMap, err := c.getDenomsFromTransactions(decodedTx)
	if err != nil {
		log.Println("Extracting denoms failed:", err.Error())
	} else {
		transaction.Metadata = ibcMap
	}

	if getter, ok := decodedTx.(TxProtoGetter); ok {
		tx := getter.GetProtoTx()
		b, err := c.enccfg.Codec.MarshalJSON(tx)
		if err != nil {
			log.Println("marshaling intermediate JSON failed: ", err.Error())
		}
		err = json.Unmarshal(b, &transaction.Tx)
		if err != nil {
			log.Println("unmarshaling intermediate JSON failed: ", err.Error())
		}
		transaction.Raw = ""
	}
	return transaction
}

func (c *rpc) translateBlock(block *tmtypes.Block) *types.Block {
	blockProto, err := block.ToProto()
	if err != nil {
		panic(err)
	}

	log.Println("Block: ", block.Hash().String(), block.Height)
	return &types.Block{
		Nonce: fmt.Sprint(c.counter.Add(1)),
		Block: blockProto,
	}
}
