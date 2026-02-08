package config

import (
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

const (
	tssConfigKey = "tss"
)

type TSSConfiger interface {
	TSSConfig() *TSSConfig
}

type TSSConfig struct {
	BinaryPath       string `fig:"binary_path,required"`
	ConfigPath       string `fig:"config_path,required"`
	CertificatesPath string `fig:"certificates_path,required"`
}

type tssConfiger struct {
	getter kv.Getter
	once   comfig.Once
}

func NewTSSConfiger(getter kv.Getter) TSSConfiger {
	return &tssConfiger{
		getter: getter,
	}
}

func (t *tssConfiger) TSSConfig() *TSSConfig {
	return t.once.Do(func() interface{} {
		var cfg TSSConfig

		if err := figure.Out(&cfg).From(kv.MustGetStringMap(t.getter, tssConfigKey)).Please(); err != nil {
			panic(errors.Wrap(err, "failed to figure out tss config"))
		}
		return &cfg
	}).(*TSSConfig)
}
