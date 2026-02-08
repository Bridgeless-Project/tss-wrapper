package observer

import "github.com/pkg/errors"

const (
	eventStartedNewEpoch = "STARTED_NEW_EPOCH"
)

var skippedEpoch = errors.New("skipped")
