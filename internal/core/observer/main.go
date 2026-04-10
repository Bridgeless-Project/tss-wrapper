package observer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	db "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/helpers"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	grpcTypes "github.com/Bridgeless-Project/tss-wrapper-svc/resources/types"
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
	tasksDb db.TasksQ
}

func New(client *http.HTTP, updaterChan chan<- types.Task, logger *logan.Entry, blockDb db.BlocksQ, tasksDb db.TasksQ) *Observer {
	retrier := helpers.NewRetrier(logger, 5, 1*time.Second)

	return &Observer{
		client:          client,
		pollingInterval: 1 * time.Second, // default polling interval
		updaterChan:     updaterChan,
		retrier:         retrier,
		events:          make(map[string]types.Task),
		logger:          logger.WithField("component", "observer"),
		blockDb:         blockDb,
		tasksDb:         tasksDb,
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
				//TODO: unlock
				o.logger.WithField("currentHeight", currentHeight).Debug("waiting for next block")
				continue
			}

			if err = o.handleBlock(ctx, &startHeight); err != nil {
				o.logger.WithError(err).
					WithField("blockNumber", startHeight).
					Error(fmt.Sprintf("failed to handle block %d", startHeight))
				continue
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
			fmt.Println("Error getting block results: ", err)
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
// saves tasks to database, and sends them to the scheduler channel
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

				// Save task to database with status Created
				taskData, err := task.MarshalData()
				if err != nil {
					return errors.Wrap(err, "failed to marshal task data")
				}

				taskID, err := o.tasksDb.Insert(db.TaskRecord{
					TaskType: task.GetTaskType(),
					Status:   grpcTypes.ProcessStatus_PROCESS_STATUS_CREATED,
					Data:     taskData,
				})
				if err != nil {
					return errors.Wrap(err, "failed to save task to database")
				}

				task.SetID(taskID)
				o.logger.WithField("task_id", taskID).
					WithField("task_type", task.GetTaskType()).
					Info("created new task")

				o.updaterChan <- task
			}
		}
	}

	return nil
}
