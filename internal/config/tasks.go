package config

import (
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

const eventsConfigKey = "events"

// TaskType represents the type of task to create for an event
type TaskType string

const (
	TaskTypeAutoResharing TaskType = "auto_resharing"
	TaskTypeUpdate        TaskType = "update"
)

type EventsConfiger interface {
	EventsConfig() []EventConfig
}

// EventConfig represents a single event-to-task mapping
type EventConfig struct {
	Event    string   `fig:"event,required"`
	TaskType TaskType `fig:"task_type,required"`
}

// eventsConfigRaw is used for parsing the config file
type eventsConfigRaw struct {
	List []EventConfig `fig:"list,required"`
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

func (c *eventsConfig) EventsConfig() []EventConfig {
	return c.once.Do(func() interface{} {
		raw := kv.MustGetStringMap(c.getter, eventsConfigKey)
		config := new(eventsConfigRaw)
		err := figure.Out(config).With(figure.BaseHooks).From(raw).Please()
		if err != nil {
			panic(errors.Wrap(err, "failed to figure out events config"))
		}
		return config.List
	}).([]EventConfig)
}
