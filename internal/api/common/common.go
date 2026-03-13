package common

import (
	db "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data"
	types "github.com/Bridgeless-Project/tss-wrapper-svc/resources"
	types2 "github.com/Bridgeless-Project/tss-wrapper-svc/resources/types"
)

func ToGetTasksResponse(records []db.TaskRecord) *types.GetTasksResponse {
	mapped := make([]*types2.TaskRecord, len(records))
	for i := range records {
		mapped[i] = ToResponseTaskRecord(records[i])
	}
	return &types.GetTasksResponse{Records: mapped}
}

func ToResponseTaskRecord(record db.TaskRecord) *types2.TaskRecord {
	var errMessage string
	if record.Error != nil {
		errMessage = *record.Error
	}
	return &types2.TaskRecord{
		Id:       record.ID,
		Tasktype: record.TaskType,
		Status:   record.Status,
		Data:     record.Data,
		Error:    errMessage,
	}
}
