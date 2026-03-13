package grpc

import (
	"context"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/common"
	apiCtx "github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/grpc/ctx"
	types2 "github.com/Bridgeless-Project/tss-wrapper-svc/resources"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc/codes"
)

func (i Implementation) GetTasks(ctx context.Context, request *types2.GetTasksRequest) (*types2.GetTasksResponse, error) {
	var (
		logger = apiCtx.Logger(ctx)
		db     = apiCtx.DB(ctx)
	)
	if request.Status != nil {
		db = db.FilterByStatus(request.GetStatus())
	}
	tasks, err := db.Page(common.GetPaginationSettings(request.Limit, request.Offset)).OrderByCreatedAt().GetAll()
	if err != nil {
		logger.WithError(err).Error("failed to fetch task data from db")
		return nil, status.Error(codes.Internal, "failed to get task details")
	}
	if tasks == nil {
		return nil, status.Error(codes.NotFound, "tasks not found")
	}
	return common.ToGetTasksResponse(tasks), nil

}
