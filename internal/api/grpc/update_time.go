package grpc

import (
	"context"
	"time"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/common"
	types2 "github.com/Bridgeless-Project/tss-wrapper-svc/resources"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i Implementation) UpdateTime(ctx context.Context, request *types2.UpdateTimeRequest) (*types2.UpdateTimeResponse, error) {
	t := time.Unix(0, request.Timestamp)
	timeTarget := time.Now().Add(time.Second * common.DefaultExecutionTime)
	if t.Before(timeTarget) {
		return nil, status.Error(codes.FailedPrecondition, "time target is in the future")
	}
	return common.ToUpdateTimeResponse(false), nil
}
