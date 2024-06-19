package cmd

import (
	"context"
	"gitlab.com/syntropynet/amberdm/publisher/dydx-publisher/internal/dydx"
	"log"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/syntropynet/data-layer-sdk/pkg/service"
)

var (
	flagTendermintAPI *string
	flagRPCAPI        *string
	flagGRPCAPI       *string
)

// startCmd represents the nft command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		name, _ := cmd.Flags().GetString("name")
		publisher := dydx.New(
			service.WithContext(ctx),
			service.WithName(name),
			service.WithPrefix(*flagPrefixName),
			service.WithNats(natsConnection),
			service.WithUserCreds(*flagUserCreds),
			service.WithNKeySeed(*flagNkey),
			service.WithPemPrivateKey(*flagPemFile),
			service.WithVerbose(*flagVerbose),
			dydx.WithTendermintAPI(*flagTendermintAPI),
			dydx.WithRPCAPI(*flagRPCAPI),
			dydx.WithGRPCAPI(*flagGRPCAPI),
		)

		if publisher == nil {
			return
		}

		pubCtx := publisher.Start()
		defer publisher.Close()

		select {
		case <-ctx.Done():
			log.Println("Shutdown")
		case <-pubCtx.Done():
			log.Println("Publisher stopped with cause: ", context.Cause(pubCtx).Error())
		}
	},
}

func setDefault(field string, value string) {
	if os.Getenv(field) == "" {
		os.Setenv(field, value)
	}
}

func init() {
	rootCmd.AddCommand(startCmd)

	const (
		DYDX_TENDERMINT = "DYDX_TENDERMINT"
		DYDX_RPC        = "DYDX_RPC"
		DYDX_GRPC       = "DYDX_GRPC"
		DYDX_NAME       = "DYDX_SUBJECT"
	)

	setDefault(DYDX_TENDERMINT, "tcp://localhost:26657")
	setDefault(DYDX_RPC, "http://localhost:1317")
	setDefault(DYDX_GRPC, "localhost:9090")
	setDefault(DYDX_NAME, "dydx")

	startCmd.Flags().StringP("name", "", os.Getenv(DYDX_NAME), "NATS subject name as in {prefix}.{name}.>")
	flagTendermintAPI = startCmd.Flags().StringP("tendermint-api", "t", os.Getenv(DYDX_TENDERMINT), "Full address to the Tendermint RPC")
	flagRPCAPI = startCmd.Flags().StringP("app-api", "a", os.Getenv(DYDX_RPC), "Full address to the Applications RPC")
	flagGRPCAPI = startCmd.Flags().StringP("grpc-api", "g", os.Getenv(DYDX_GRPC), "Full address to the Applications gRPC")
}
