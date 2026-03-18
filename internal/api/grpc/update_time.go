package grpc

import (
	"context"
	"time"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/common"
	apiCtx "github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/grpc/ctx"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/tss/timechanger"
	types2 "github.com/Bridgeless-Project/tss-wrapper-svc/resources"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i Implementation) UpdateTime(ctx context.Context, request *types2.UpdateTimeRequest) (*types2.UpdateTimeResponse, error) {
	var (
		scheduler = apiCtx.Scheduler(ctx)
		tssConfig = apiCtx.TSSConfig(ctx)
	)
	t := time.Unix(request.Timestamp, 0).UTC()
	timeTarget := time.Now().UTC().Add(time.Second * common.DefaultExecutionTime)
	if t.Before(timeTarget) {
		return nil, status.Error(codes.FailedPrecondition, "time target is in the future")
	}
	task := timechanger.NewTask(tssConfig)
	task.StartTime = t
	scheduler.ScheduleTask(ctx, task)
	return common.ToUpdateTimeResponse(false), nil
}
