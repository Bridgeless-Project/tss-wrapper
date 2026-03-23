package grpc

import (
	"context"
	"time"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/common"
	apiCtx "github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/grpc/ctx"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/tss/timechanger"
	types "github.com/Bridgeless-Project/tss-wrapper-svc/resources"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i Implementation) UpdateTime(ctx context.Context, request *types.UpdateTimeRequest) (*types.UpdateTimeResponse, error) {
	var (
		scheduler = apiCtx.Scheduler(ctx)
	)

	if !common.ValidateTime(request) {
		return nil, status.Error(codes.FailedPrecondition, "timestamp is invalid")
	}

	task := timechanger.NewTask(apiCtx.TSSConfig(ctx))
	task.StartTime = time.Unix(request.StartTime, 0).Local()
	task.TargetTime = time.Unix(request.TargetTime, 0).Local()

	scheduler.ScheduleTask(context.WithoutCancel(ctx), task)
	return common.ToUpdateTimeResponse(), nil
}
