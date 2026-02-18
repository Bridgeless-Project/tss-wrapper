package config

import (
	"crypto/tls"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

const (
	tendermintConnectorKey = "tendermint_connector"
)

type TendermintConnector interface {
	TendermintHttpClient() *http.HTTP
	TendermintGrpcClient() *grpc.ClientConn
}

type tenderminter struct {
	getter kv.Getter
	once   comfig.Once
}

func (t *tenderminter) TendermintHttpClient() *http.HTTP {
	cfg := t.config()
	client, err := http.New(cfg.RPC, "/websocket")
	if err != nil {
		panic(errors.Wrap(err, "failed to create tendermint http client"))
	}

	if err = client.Start(); err != nil {
		panic(errors.Wrap(err, "failed to start tendermint http client"))
	}

	return client
}

func (t *tenderminter) TendermintGrpcClient() *grpc.ClientConn {
	cfg := t.config()

	tlsConfig := &tls.Config{
		InsecureSkipVerify: isHTTPS(cfg.GRPC),
	}

	con, err := grpc.Dial(cfg.GRPC, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)), grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:    10 * time.Second,
		Timeout: 20 * time.Second,
	}))
	if err != nil {
		panic(errors.Wrap(err, "failed to create tendermint grpc client"))
	}

	return con
}

func NewTendermintConnector(getter kv.Getter) TendermintConnector {
	return &tenderminter{
		getter: getter,
	}
}

type tenderminterCfg struct {
	RPC  string `fig:"tendermint_rpc,required"`
	GRPC string `fig:"tendermint_grpc,required"`
}

func (t *tenderminter) config() *tenderminterCfg {
	return t.once.Do(func() interface{} {
		var cfg tenderminterCfg

		if err := figure.Out(&cfg).From(kv.MustGetStringMap(t.getter, tendermintConnectorKey)).Please(); err != nil {
			panic(errors.Wrap(err, "failed to figure out tendermint connector config"))
		}
		return &cfg
	}).(*tenderminterCfg)
}

func isHTTPS(domen string) bool {
	ok, err := regexp.Match("https:", []byte(domen))
	if err != nil || !ok {
		return false
	}

	return true
}
