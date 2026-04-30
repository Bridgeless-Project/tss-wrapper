package migration

import (
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/helpers"
	"github.com/pkg/errors"
)

func (t *Task) updateConfigBeforeExecution() error {
	configer := helpers.NewConfigManager(t.ConfigPath)
	if err := configer.Load(); err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	return nil
}

func (t *Task) updateConfigAfterExecution() error {
	// TODO:
	// - rollback btc wallets
	// - update start time

	return nil
}
