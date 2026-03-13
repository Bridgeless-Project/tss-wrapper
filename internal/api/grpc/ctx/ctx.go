package ctx

import (
	"context"

	db "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data"
	"gitlab.com/distributed_lab/logan/v3"
)

type ctxKey int

const (
	dbKey          ctxKey = iota
	loggerKey      ctxKey = iota
	clientsRepoKey ctxKey = iota
	broadcasterKey ctxKey = iota
	connectorKey   ctxKey = iota
)

func DBProvider(q db.TasksQ) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {

		return context.WithValue(ctx, dbKey, q)
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
