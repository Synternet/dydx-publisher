package dydx

import (
	"time"

	"github.com/synternet/data-layer-sdk/pkg/options"
	"github.com/synternet/data-layer-sdk/pkg/service"
)

var (
	TendermintAPIParam = "tm"
	RPCAPIParam        = "rpc"
	GRPCAPIParam       = "grpc"
	MempoolPeriodParam = "mmp"
)

func WithTendermintAPI(url string) options.Option {
	return func(o *options.Options) {
		service.WithParam(TendermintAPIParam, url)(o)
	}
}

func (p *Publisher) TendermintApi() string {
	return options.Param(p.Options, TendermintAPIParam, "tcp://localhost:26657")
}

func WithRPCAPI(url string) options.Option {
	return func(o *options.Options) {
		service.WithParam(RPCAPIParam, url)(o)
	}
}

func (p *Publisher) RPCApi() string {
	return options.Param(p.Options, RPCAPIParam, "http://localhost:1317")
}

func WithGRPCAPI(url string) options.Option {
	return func(o *options.Options) {
		service.WithParam(GRPCAPIParam, url)(o)
	}
}

func (p *Publisher) GRPCApi() string {
	return options.Param(p.Options, GRPCAPIParam, "localhost:9090")
}

func WithMempoolPeriod(d time.Duration) options.Option {
	return func(o *options.Options) {
		service.WithParam(MempoolPeriodParam, d)(o)
	}
}

func (p *Publisher) MempoolPeriod() time.Duration {
	return options.Param(p.Options, MempoolPeriodParam, time.Millisecond*50)
}
