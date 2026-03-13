package grpc

import (
	"context"

	apiCtx "github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/grpc/ctx"
	types2 "github.com/Bridgeless-Project/tss-wrapper-svc/resources"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i Implementation) CheckUpdate(ctx context.Context, identifier *types2.CheckUpdateRequest) (*types2.CheckUpdateResponse, error) {
	var (
		_ = apiCtx.Logger(ctx)
		_ = apiCtx.DB(ctx)
	)
	return nil, status.Error(codes.Unimplemented, "not implemented")

}
