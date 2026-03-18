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
	timeStart := time.Unix(request.StartTime, 0).Local()
	timeTarget := time.Unix(request.TargetTime, 0).Local().Add(time.Second * common.DefaultExecutionTime)
	if timeStart.After(timeTarget) {
		return nil, status.Error(codes.FailedPrecondition, "time target is too early")
	}
	if timeStart.Before(time.Now().UTC()) {
		return nil, status.Error(codes.FailedPrecondition, "time start is before time.Now()")
	}
	task := timechanger.NewTask(tssConfig)
	task.StartTime = timeStart
	task.TargetTime = timeTarget
	scheduler.ScheduleTask(context.WithoutCancel(ctx), task)
	return common.ToUpdateTimeResponse(true), nil
}
