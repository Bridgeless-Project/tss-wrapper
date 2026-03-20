package ctx

import (
	"context"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/config"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/core/scheduler"

	db "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data"
	"gitlab.com/distributed_lab/logan/v3"
)

type ctxKey int

const (
	dbKey ctxKey = iota
	loggerKey
	schedulerKey
	tssConfigKey
)

func DBProvider(value db.TasksQ) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, dbKey, value)
	}
}

// DB always returns unique connection
func DB(ctx context.Context) db.TasksQ {
	return ctx.Value(dbKey).(db.TasksQ).New()
}

func LoggerProvider(l *logan.Entry) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, loggerKey, l)
	}
}
func Logger(ctx context.Context) *logan.Entry {
	return ctx.Value(loggerKey).(*logan.Entry)
}

func SchedulerProvider(value *scheduler.Scheduler) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, schedulerKey, value)
	}
}
func Scheduler(ctx context.Context) *scheduler.Scheduler {
	return ctx.Value(schedulerKey).(*scheduler.Scheduler)
}

func TSSConfigProvider(value *config.TSSConfig) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, tssConfigKey, value)
	}
}
func TSSConfig(ctx context.Context) *config.TSSConfig {
	return ctx.Value(tssConfigKey).(*config.TSSConfig)
}
