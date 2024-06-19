package dydx

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
)

const subscriberName = "dldydxpub"

func setMaxValue(a *atomic.Uint64, v uint64) {
	for {
		oldValue := a.Load()
		if oldValue >= v {
			return
		}

		if a.CompareAndSwap(oldValue, v) {
			return
		}
	}
}

func (c *rpc) bufferChannel(events <-chan ctypes.ResultEvent, size int) <-chan ctypes.ResultEvent {
	ch := make(chan ctypes.ResultEvent, size)
	c.group.Go(func() error {
		defer close(ch)
		for {
			select {
			case <-c.ctx.Done():
				log.Println("bufferChannel: Context Done")
				return nil
			case ev, ok := <-events:
				if !ok {
					log.Println("bufferChannel: events closed")
					return nil
				}

				c.evtCounter.Add(1)

				setMaxValue(&c.queueMaxSize, uint64(len(ch)))

				select {
				case ch <- ev:
				default:
					c.evtSkipCounter.Add(1)
					log.Println("bufferChannel: Overflow! Skipping event: ", ev.Query)
				}
			}
		}
	})
	return ch
}

func (c *rpc) Subscribe(ctx context.Context, publish func(msg any, suffixes ...string) error, onFail func(error)) error {
	txs, err := c.tendermint.Subscribe(ctx, subscriberName, "tm.event='Tx'")
	if err != nil {
		c.errCounter.Add(1)
		return err
	}
	c.group.Go(func() error {
		return c.handleSubscriptions(ctx, publish, c.bufferChannel(txs, 20480))
	})

	blocks, err := c.tendermint.Subscribe(ctx, subscriberName, "tm.event='NewBlock'")
	if err != nil {
		c.errCounter.Add(1)
		return err
	}
	sentinel := time.NewTimer(time.Minute)
	c.group.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-sentinel.C:
				return fmt.Errorf("block event timed out")
			}
		}
	})
	c.group.Go(func() error {
		return c.handleSubscriptions(
			ctx,
			func(msg any, suffixes ...string) error {
				sentinel.Reset(time.Minute)
				return publish(msg, suffixes...)
			},
			c.bufferChannel(blocks, 20480),
		)
	})

	return nil
}

func (c *rpc) handleSubscriptions(ctx context.Context, publish func(msg any, suffixes ...string) error, txs <-chan ctypes.ResultEvent) error {
	for {
		select {
		case <-ctx.Done():
			log.Println("handleSubscriptions: Context Done")
			return nil
		case <-c.ctx.Done():
			log.Println("handleSubscriptions: c.Context Done")
			return nil
		case tx, ok := <-txs:
			if !ok {
				log.Println("handleSubscriptions: txs closed")
				return fmt.Errorf("txs closed")
			}

			switch data := tx.Data.(type) {
			case tmtypes.EventDataNewBlock:
				c.blockCounter.Add(1)
				publish(
					c.translateBlock(data.Block),
					"block",
				)
			case tmtypes.EventDataTx:
				c.txCounter.Add(1)
				hash := hex.EncodeToString(tmtypes.Tx(data.Tx).Hash())
				tx := c.translateTransaction(data.Tx, hash, fmt.Sprint(c.counter.Add(1)), &data.TxResult, &data.TxResult.Result.Code)
				publish(
					tx,
					"tx",
				)
				log.Println("Transaction: ", tx.TxID)
			}
		}
	}
}
