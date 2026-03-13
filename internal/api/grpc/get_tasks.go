package grpc

import (
	"context"

	apiCtx "github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/grpc/ctx"
	types2 "github.com/Bridgeless-Project/tss-wrapper-svc/resources"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc/codes"
)

func (i Implementation) GetTasks(ctx context.Context, identifier *types2.GetTasksRequest) (*types2.GetTasksResponse, error) {
	var (
		_ = apiCtx.Logger(ctx)
		_ = apiCtx.DB(ctx)
	)
	return nil, status.Error(codes.Unimplemented, "not implemented")

}
