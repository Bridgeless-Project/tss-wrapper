package types

import (
	"context"
	"time"
)

type Task interface {
	GetID() int64
	GetTime() time.Time
	GetTaskType() string

	SetID(id int64)
	
	Execute(ctx context.Context) error
	Parse(attributes []Attribute) (Task, error)
	StartScheduling(ctx context.Context, c chan<- Task)

	MarshalData() (string, error)
	UnmarshalData(data string) error
}
