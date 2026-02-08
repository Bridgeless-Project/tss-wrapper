package types

import (
	"context"
	"time"
)

type Task interface {
	Execute(ctx context.Context) error
	GetTime() time.Time
	Parse(attributes []Attribute) (Task, error)
	StartScheduling(ctx context.Context, c chan<- Task)
}
