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

const defailtLimit = 15
const defailtOffset = 0

func (i Implementation) GetTasks(ctx context.Context, request *types2.GetTasksRequest) (*types2.GetTasksResponse, error) {
	var (
		logger = apiCtx.Logger(ctx)
		db     = apiCtx.DB(ctx)
	)
	if request.Status != nil {
		db = db.FilterByStatus(request.GetStatus())
	}
	var limit, offset uint64
	if request.Limit == nil {
		limit = defailtLimit
	} else {
		limit = request.GetLimit()
	}
	if request.Offset == nil {
		offset = defailtOffset
	} else {
		offset = request.GetOffset()
	}
	tasks, err := db.Page(pgdb.OffsetPageParams{Limit: limit, PageNumber: offset}).Get()
	if err != nil {
		logger.WithError(err).Error("failed to fetch task data from db")
		return nil, status.Error(codes.Internal, "failed to get task details")
	}
	return common.ToGetTasksResponse(tasks), nil

}
