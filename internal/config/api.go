package config

import (
	"net"
	"reflect"

	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

const (
	listenersKey = "listeners"
)

type Listenerer interface {
	GRPCListener() net.Listener
	HTTPListener() net.Listener
}

type listener struct {
	getter kv.Getter
	once   comfig.Once
}
type listeners struct {
	ApiGrpc net.Listener `fig:"api_grpc_addr,required"`
	ApiHttp net.Listener `fig:"api_http_addr,required"`
}

func NewListener(getter kv.Getter) Listenerer {
	return &listener{
		getter: getter,
	}
}

func (l *listener) GRPCListener() net.Listener { return l.listeners().ApiGrpc }

func (l *listener) HTTPListener() net.Listener { return l.listeners().ApiHttp }

func (l *listener) listeners() *listeners {
	return l.once.Do(func() interface{} {
		var ls *listeners
		err := figure.
			Out(&ls).
			With(figure.BaseHooks, listenerHooks).
			From(kv.MustGetStringMap(l.getter, listenersKey)).
			Please()
		if err != nil {
			panic(errors.Wrap(err, "failed to load listener config"))
		}

		return ls
	}).(*listeners)
}

var listenerHooks = figure.Hooks{
	"net.Listener": func(value interface{}) (reflect.Value, error) {
		switch addr := value.(type) {
		case string:
			ls, err := net.Listen("tcp", addr)
			if err != nil {
				return reflect.Value{}, errors.Wrapf(err, "failed to listen on %s", addr)
			}

			return reflect.ValueOf(ls), nil
		default:
			return reflect.Value{}, errors.Errorf("unsupported conversion from %T", value)
		}
	},
}
