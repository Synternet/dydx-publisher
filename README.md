# dYdX Publisher

[![Latest release](https://img.shields.io/github/v/release/synternet/dydx-publisher)](https://github.com/synternet/dydx-publisher/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub Workflow Status (with event)](https://img.shields.io/github/actions/workflow/status/synternet/dydx-publisher/github-ci.yml?label=github-ci)](https://github.com/synternet/dydx-publisher/actions/workflows/github-ci.yml)

Establishes connection with dYdX node and publishes dYdx blockchain data to Synternet Data Layer via NATS connection.

## Usage

Building from source.

```bash
make build
```

Getting usage help.

```bash
./build/dydx-publisher --help
```

Running executable with flags.

```bash
./build/dydx-publisher \
  --nats-url nats://dal-broker \
  --prefix my-org \
  --nats-nkey SA..BC \
  --nats-jwt eyJ0e...aW \
  start \
  --app-api http://localhost:1317 \
  --grpc-api localhost:9090 \
  --tendermint-api tcp://localhost:26657 \
  --publisher-name dydx
```

Running executable with environment variables. Environment variables are automatically attempted to be loaded from `.env` file.
Any flag can be used as environment variables by updating flag to be `UPPERCASE` words separated by `_` (e.g.: flag `nats-nkey` == env var `NATS_NKEY`).

```bash
./build/dydx-publisher start

// .env file content
NATS_URL=nats://dal-broker
PREFIX=my-org
NATS_NKEY=SA..BC
NATS_JWT=eyJ0e...aW
APP_API=http://localhost:1317
GRPC_API=localhost:9090
TENDERMINT_API=tcp://localhost:26657
PUBLISHER_NAME=dydx
```

Note: instead of user `NATS_NKEY` and `NATS_JWT` single value of `NATS_ACC_NKEY` can be supplied. In Synternet Data Layer Developer Portal
this is called `Access Token`. See [here](https://docs.synternet.com/build/data-layer/developer-portal/data-layer-authentication#access-token) for more details.

## Things to consider

- dYdX gRPC should be configured at 9090 port
- gRPC endpoint is HTTP/2, thus any proxies or load balancers should be configured appropriately

## Telemetry

dYdX publisher sends telemetry data regularly on `{prefix}.{name}.telemetry` subject. The contents of this message look something like this:

```json
{"nonce":"207aa","status":{"blocks":1,"errors":0,"events":{"max_queue":40,"queue":1,"skipped":0,"total":7},"goroutines":39,"indexer":{"blocks_per_hour":1142,"errors":0,"ibc":{"cache_misses":9,"tokens":916},"pool":{"current_height":15040158,"sync_count":0}},"mempool.txs":8,"messages":{"bytes_in":0,"bytes_out":496916,"in":0,"out":17,"out_queue":0,"out_queue_cap":1000},"period":"3.000121219s","pools":0,"published":0,"txs":6,"unknown_events":0,"uptime":"110h51m42.00052816s"}}
```

You can configure the interval of these messages by setting `TELEMETRY_PERIOD` environment variable(default is `"3s"`).

## Docker

### Build from source

1. Build image.

```bash
docker build -f ./docker/Dockerfile -t dydx-publisher .
```

2. Run container with passed environment variables. See [entrypoint.sh](./docker/entrypoint.sh) for available env variables in container.

```bash
docker run -it --rm --env-file=.env dydx-publisher
```

### Prebuilt image

Run container with passed environment variables.

```bash
docker run -it --rm --env-file=.env ghcr.io/synternet/dydx-publisher:latest
```

### Docker Compose

`docker-compose.yml` file.

```yaml
version: '3.8'

services:
  dydx-publisher:
    image: ghcr.io/synternet/dydx-publisher:latest
    environment:
      - NATS_URL=nats://dal-broker
      - PREFIX=my-org
      - NATS_NKEY=SA..BC
      - NATS_JWT=eyJ0e...aW
      - APP_API=http://localhost:1317
      - GRPC_API=localhost:9090
      - TENDERMINT_API=tcp://localhost:26657
      - PUBLISHER_NAME=dydx
```

## dYdX Full Node

You can refer to the official [documentation](https://docs.dydx.exchange/infrastructure_providers-validators/running_full_node) for instructions how to run a full node in Mainnet.

### Hardware

The minimum recommended specs for running a node is the following:

16-core, x86_64 architecture processor
64 GiB RAM
500 GiB of locally attached SSD storage

## Contributing

We welcome contributions from the community. Whether it's a bug report, a new feature, or a code fix, your input is valued and appreciated.

## Synternet

If you have any questions, ideas, or simply want to connect with us, we encourage you to reach out through any of the following channels:

- **Discord**: Join our vibrant community on Discord at [https://discord.com/invite/Ze7Kswye8B](https://discord.com/invite/Ze7Kswye8B). Engage in discussions, seek assistance, and collaborate with like-minded individuals.
- **Telegram**: Connect with us on Telegram at [https://t.me/synternet](https://t.me/synternet). Stay updated with the latest news, announcements, and interact with our team members and community.
- **Email**: If you prefer email communication, feel free to reach out to us at devrel@synternet.com. We're here to address your inquiries, provide support, and explore collaboration opportunities.
