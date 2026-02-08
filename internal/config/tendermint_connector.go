package config

import (
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

const (
	tendermintConnectorKey = "tendermint_connector"
)

type TendermintConnector interface {
	TendermintHttpClient() *http.HTTP
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

func NewTendermintConnector(getter kv.Getter) TendermintConnector {
	return &tenderminter{
		getter: getter,
	}
}

type tenderminterCfg struct {
	RPC string `fig:"tendermint_rpc,required"`
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
