package observer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	db "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/helpers"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	"github.com/pkg/errors"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"gitlab.com/distributed_lab/logan/v3"

	"github.com/tendermint/tendermint/rpc/client/http"
)

type Observer struct {
	client          *http.HTTP
	pollingInterval time.Duration
	// map of event name to appropriate Task type
	events      map[string]types.Task
	updaterChan chan<- types.Task

	logger  *logan.Entry
	retrier helpers.Retrier
	blockDb db.BlocksQ
}

func New(client *http.HTTP, updaterChan chan<- types.Task, logger *logan.Entry, blockDb db.BlocksQ) *Observer {
	retrier := helpers.NewRetrier(logger, 0, 1*time.Second)

	return &Observer{
		client:          client,
		pollingInterval: 1 * time.Second, // default polling interval
		updaterChan:     updaterChan,
		retrier:         retrier,
		events:          make(map[string]types.Task),
		logger:          logger.WithField("component", "observer"),
		blockDb:         blockDb,
	}
}

func (o *Observer) WithEvent(event string, eventType types.Task) *Observer {
	o.events[event] = eventType
	return o
}

func (o *Observer) WithPollingInterval(interval time.Duration) *Observer {
	o.pollingInterval = interval
	return o
}

func (o *Observer) Run(ctx context.Context, startHeight int64) error {
	ticker := time.NewTicker(o.pollingInterval)
	defer ticker.Stop()

	if startHeight == 0 {
		latestHeight, err := o.getCurrentHeight(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to get current height")
		}
		startHeight = latestHeight
	}

	if err := o.blockDb.Insert(db.LatestBlock{BlockId: startHeight}); err != nil {
		return errors.Wrap(err, "failed to insert latest block")
	}
	for {
		select {
		case <-ctx.Done():
			// graceful shutdown

			return nil
		case <-ticker.C:
			currentHeight, err := o.getCurrentHeight(ctx)
			if err != nil {
				o.logger.WithError(err).Error("failed to get current height")
				continue
			}

			if startHeight > currentHeight {
				o.logger.WithField("currentHeight", currentHeight).Debug("waiting for next block")
				continue
			}

			if err = o.handleBlock(ctx, &startHeight); err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to handle block %d", startHeight))
			}

			if err = o.blockDb.UpdateLatestBlockId(db.LatestBlock{BlockId: startHeight}); err != nil {
				o.logger.WithError(err).
					WithField("blockNumber", startHeight).
					Error("failed to update latest block height, will retry")
				continue
			}

			startHeight++
		}
	}

}

func (o *Observer) getCurrentHeight(ctx context.Context) (int64, error) {
	var info *coretypes.ResultABCIInfo

	getCurrentHeight := func() error {
		var err error
		info, err = o.client.ABCIInfo(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to get ABCI info")
		}
		return nil
	}

	if err := o.retrier.Do(ctx, getCurrentHeight); err != nil {
		return 0, errors.Wrap(err, "failed to get current height")
	}

	return info.Response.LastBlockHeight, nil
}

func (o *Observer) handleBlock(ctx context.Context, height *int64) error {
	var blockResult *coretypes.ResultBlockResults

	getBlockResult := func() error {
		var err error
		blockResult, err = o.client.BlockResults(ctx, height)
		if err != nil {
			return errors.Wrap(err, "failed to get block results")
		}

		return nil
	}

	if err := o.retrier.Do(ctx, getBlockResult); err != nil {
		return errors.Wrap(err, "failed to get block results")
	}

	err := o.handleEventFromTxResults(blockResult.TxsResults)
	if err != nil {
		return errors.Wrap(err, "failed to handle events from tx results")
	}
	return nil
}

// handleEventFromTxResults parses transaction logs, extracts events,
// and sends matching tasks to the scheduler channel
func (o *Observer) handleEventFromTxResults(txs []*abciTypes.ResponseDeliverTx) error {
	for _, tx := range txs {
		var msgs []types.MsgEvent

		if tx.Log == "" || !json.Valid([]byte(tx.Log)) {
			continue
		}

		if err := json.Unmarshal([]byte(tx.Log), &msgs); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to unmarshal log: %v", tx.Log))
		}
		for _, msg := range msgs {
			for _, event := range msg.Events {
				taskType, ok := o.events[event.Type]
				if !ok {
					continue
				}
				task, err := taskType.Parse(event.Attributes)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("Failed to parse event attributes: %v", event.Attributes))
				}

				o.updaterChan <- task
			}
		}

	}

	return nil
}
