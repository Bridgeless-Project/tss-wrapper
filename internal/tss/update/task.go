package autoreshering

import (
	"fmt"
	"time"

	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
)

type Task struct {
	link    string // link to the tss binary to download
	version string
}

func NewTask() *Task {
	return &Task{}
}

func (a *Task) Execute() error {
	return nil
}

func (a *Task) GetTime() time.Time {
	return time.Now()
}

func (a *Task) Parse(attributes []types.Attribute) error {
	for _, attribute := range attributes {

		switch attribute.Key {

		case bridgeTypes.AttributeKeyCommissionAmount:
			deposit.CommissionAmount = attribute.Value
		default:
			return errors.Wrap(errors.New(fmt.Sprintf("unknown attribute key: %s", attribute.Key)), "failed to parse attribute")
		}
	}
	return nil
}

func (a *Task) Name() string {
	return "AutoResheringTask"
}

func (a *Task) DownloadTSSBinary() error {
	// Logic to download the TSS binary from the specified link
	return nil
}
