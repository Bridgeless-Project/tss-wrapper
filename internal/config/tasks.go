package config

import (
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

const eventsConfigKey = "events"

type EventsConfiger interface {
	EventsConfig() *EventsConfig
}

type EventsConfig struct {
	Event string     `fig:"event,required"`
	Task  types.Task `fig:"task,required"`
}

func NewEventsConfiger(getter kv.Getter) EventsConfiger {
	return &eventsConfig{
		getter: getter,
	}
}

type eventsConfig struct {
	getter kv.Getter
	once   comfig.Once
}

func (c *eventsConfig) EventsConfig() *EventsConfig {
	return c.once.Do(func() interface{} {
		raw := kv.MustGetStringMap(c.getter, eventsConfigKey)
		config := new(EventsConfig)
		err := figure.Out(config).With(figure.BaseHooks).From(raw).Please()
		if err != nil {
			panic(errors.Wrap(err, "failed to figure out"))
		}
		return config
	}).(*EventsConfig)
}
