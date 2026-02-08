package helpers

import (
	"context"
	"time"

	"github.com/avast/retry-go"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type Retrier struct {
	logger       *logan.Entry
	retries      uint
	retryTimeout time.Duration
}

func NewRetrier(logger *logan.Entry, retries uint, retryTime time.Duration) Retrier {
	return Retrier{
		retries:      retries,
		logger:       logger,
		retryTimeout: retryTime,
	}
}

func (e Retrier) Do(ctx context.Context, function func() error) error {
	err := retry.Do(
		function,
		retry.Attempts(e.retries),
		retry.Delay(e.retryTimeout),
		retry.DelayType(retry.BackOffDelay),
		retry.Context(ctx),
		retry.OnRetry(func(n uint, err error) {
			e.logger.WithError(err).WithField("attempt", n+1).Warn("retrying step")
		}),
	)

	return errors.Wrap(err, "failed to execute function")
}
