package grpc

import (
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/grpc/types"
)

var _ types.APIServer = Implementation{}

type Implementation struct{}
