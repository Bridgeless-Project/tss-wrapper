package grpc

import (
	"context"

	types2 "github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/grpc/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i Implementation) CheckUpdate(ctx context.Context, identifier *types2.CheckUpdateRequest) (*types2.CheckUpdateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
