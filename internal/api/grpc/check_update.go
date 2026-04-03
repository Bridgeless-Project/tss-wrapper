package grpc

import (
	"context"

	types "github.com/Bridgeless-Project/tss-wrapper-svc/resources"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i Implementation) CheckUpdate(ctx context.Context, identifier *types.CheckUpdateRequest) (*types.CheckUpdateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
