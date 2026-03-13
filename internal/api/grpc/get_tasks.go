package grpc

import (
	"context"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/common"
	apiCtx "github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/grpc/ctx"
	types2 "github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/grpc/types"
	"gitlab.com/distributed_lab/kit/pgdb"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc/codes"
)

func (i Implementation) GetTasks(ctx context.Context, request *types2.GetTasksRequest) (*types2.GetTasksResponse, error) {
	var (
		logger = apiCtx.Logger(ctx)
		db     = apiCtx.DB(ctx)
	)
	tasks, err := db.FilterByStatus(request.GetStatus()).Page(pgdb.OffsetPageParams{Limit: request.GetLimit(), PageNumber: request.GetPages()}).Get()
	if err != nil {
		logger.WithError(err).Error("failed to fetch task data from db")
		return nil, status.Error(codes.Internal, "failed to get task details")
	}
	return common.ToGetTasksResponse(tasks), nil

}
