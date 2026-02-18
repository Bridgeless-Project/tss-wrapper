package helpers

import (
	"context"
	
	bridgetypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/cosmos/gogoproto/grpc"
)

func GetEpoch(ctx context.Context, connection grpc.ClientConn) (uint32, error) {
	params, err := bridgetypes.NewQueryClient(connection).Params(ctx, new(bridgetypes.QueryParamsRequest))
	if err != nil {
		return 0, err
	}

	return params.Params.Epoch, nil
}
