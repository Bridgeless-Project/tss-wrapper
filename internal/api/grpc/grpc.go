package grpc

import (
	types "github.com/Bridgeless-Project/tss-wrapper-svc/resources"
)

var _ types.APIServer = Implementation{}

type Implementation struct{}
