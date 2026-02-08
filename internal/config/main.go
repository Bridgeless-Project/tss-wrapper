package config

import (
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/kit/pgdb"
)

type Config interface {
	comfig.Logger
	pgdb.Databaser
	Listenerer
	EventsConfiger
	TendermintConnector
	TSSConfiger
}

type config struct {
	getter kv.Getter
	comfig.Logger
	pgdb.Databaser
	Listenerer
	EventsConfiger
	TendermintConnector
	TSSConfiger
}

func New(getter kv.Getter) Config {
	return config{
		getter:              getter,
		Logger:              comfig.NewLogger(getter, comfig.LoggerOpts{}),
		Databaser:           pgdb.NewDatabaser(getter),
		Listenerer:          NewListener(getter),
		EventsConfiger:      NewEventsConfiger(getter),
		TendermintConnector: NewTendermintConnector(getter),
		TSSConfiger:         NewTSSConfiger(getter),
	}
}
