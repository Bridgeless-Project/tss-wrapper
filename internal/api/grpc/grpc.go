package grpc

import (
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/types"
)

var _ types.APIServer = Implementation{}

type Implementation struct{}
