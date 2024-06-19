package dydx

import (
	"context"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	IBCTypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	"github.com/dydxprotocol/v4-chain/protocol/app"
	"github.com/synternet/dydx-publisher/pkg/types"
	"golang.org/x/sync/errgroup"

	tmlog "github.com/cometbft/cometbft/libs/log"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"google.golang.org/grpc"
)

type rpc struct {
	ctx           context.Context
	group         *errgroup.Group
	cancel        context.CancelCauseFunc
	counter       atomic.Int64
	tendermintUrl string
	rpcUrl        string
	tendermint    *rpchttp.HTTP
	rpc           *http.Client
	grpc          *grpc.ClientConn
	mempoolSet    map[string]struct{}
	// marshaler      *codec.ProtoCodec
	enccfg         app.EncodingConfig
	ibcQueryClient IBCTypes.QueryClient
	ibcTraceCache  map[string]IBCTypes.DenomTrace

	blockCounter    atomic.Uint64
	txCounter       atomic.Uint64
	errCounter      atomic.Uint64
	evtCounter      atomic.Uint64
	evtSkipCounter  atomic.Uint64
	evtOtherCounter atomic.Uint64
	ibcMisses       atomic.Uint64
	queueMaxSize    atomic.Uint64
	maxQueueSize    uint64
}

func newRpc(ctx context.Context, cancel context.CancelCauseFunc, group *errgroup.Group, tendermintUrl, rpcUrl string, grpcUrl string) (*rpc, error) {
	ret := &rpc{
		ctx:           ctx,
		group:         group,
		cancel:        cancel,
		tendermintUrl: tendermintUrl,
		rpcUrl:        rpcUrl,
		rpc:           &http.Client{},
		mempoolSet:    make(map[string]struct{}),
	}

	log.Printf("Using tendermint=%s rpc=%s grpc=%s\n", tendermintUrl, rpcUrl, grpcUrl)

	client, err := rpchttp.NewWithTimeout(tendermintUrl, "/websocket", 20)
	if err != nil {
		return nil, err
	}
	tmlog.AllowAll()

	err = client.Start()
	if err != nil {
		return nil, err
	}
	ret.tendermint = client

	enccfg := app.GetEncodingConfig()
	ret.enccfg = enccfg

	ret.ibcTraceCache = make(map[string]IBCTypes.DenomTrace)

	grpcConn, err := grpc.Dial(
		grpcUrl,
		grpc.WithInsecure())

	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}
	ret.grpc = grpcConn
	ret.ibcQueryClient = IBCTypes.NewQueryClient(ret.grpc)
	ret.preHeatDenomTraceCache()

	return ret, nil
}

func (c *rpc) Close() error {
	log.Println("Publisher.RPC.Close")
	c.cancel(nil)
	var err []error
	if c.tendermint != nil {
		log.Println("Unsubscribe All")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		err = append(err, c.tendermint.UnsubscribeAll(ctx, subscriberName))
		log.Println("Tendermint Stop")
		err = append(err, c.tendermint.Stop())
	}
	log.Println("Waiting on group")
	errGr := c.group.Wait()
	if !errors.Is(errGr, context.Canceled) {
		err = append(err, errGr)
	}
	log.Println("Publisher.RPC.Close DONE")
	return errors.Join(err...)
}

func (c *rpc) ChainID(ctx context.Context) (string, error) {
	info, err := c.tendermint.Block(ctx, nil)
	if err != nil {
		c.errCounter.Add(1)
		return "", err
	}

	return info.Block.ChainID, nil
}

func (c *rpc) Mempool() ([]*types.Transaction, error) {
	var limit int = 100
	res, err := c.tendermint.UnconfirmedTxs(c.ctx, &limit)
	if err != nil {
		c.errCounter.Add(1)
		return nil, err
	}
	if res.Count == 0 {
		return nil, nil
	}

	// NOTE: This should never be called asynchronously, therefore no need to synchronize
	currentSet := make(map[string]struct{}, res.Count)

	txs := make([]*types.Transaction, 0, res.Count)
	for _, tx := range res.Txs {
		hash := hex.EncodeToString(tx.Hash())
		currentSet[hash] = struct{}{}
		if _, ok := c.mempoolSet[hash]; ok {
			continue
		}
		c.mempoolSet[hash] = struct{}{}

		res := c.translateTransaction(tx, hash, "", nil, nil)
		txs = append(txs, res)

		log.Println("Mempool: ", hash)
	}
	// Remove hashes from mempoolSet that were not observed in the mempool this time.
	// That means that the tx was removed from the mempool.
	for k := range c.mempoolSet {
		if _, ok := currentSet[k]; ok {
			continue
		}
		delete(c.mempoolSet, k)
	}

	if len(txs) == 0 {
		return nil, nil
	}

	return txs, nil
}

func (p *rpc) getStatus() map[string]string {
	queueSize := p.queueMaxSize.Swap(0)
	if queueSize > p.maxQueueSize {
		p.maxQueueSize = queueSize
	}

	return map[string]string{
		"ibc_tokens":       string(len(p.ibcTraceCache)),
		"ibc_cache_misses": string(p.ibcMisses.Load()),
		"blocks":           string(p.blockCounter.Swap(0)),
		"txs":              string(p.txCounter.Swap(0)),
		"errors":           string(p.errCounter.Swap(0)),
		"event_total":      string(p.evtCounter.Swap(0)),
		"event_other":      string(p.evtOtherCounter.Swap(0)),
		"event_skipped":    string(p.evtSkipCounter.Load()),
		"event_queue":      string(queueSize),
		"event_max_queue":  string(p.maxQueueSize),
	}
}
