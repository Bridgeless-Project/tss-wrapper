package helpers

import (
	"context"

	bridgetypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/cosmos/gogoproto/grpc"
	"github.com/pkg/errors"
)

func GetEpochState(ctx context.Context, epochID uint32, connection grpc.ClientConn) (*bridgetypes.Epoch, error) {
	epoch, err := bridgetypes.NewQueryClient(connection).GetEpochById(ctx, &bridgetypes.QueryGetEpoch{
		EpochId: epochID,
	})
	if err != nil {
		if errors.Is(err, bridgetypes.ErrEpochNotFound.GRPCStatus().Err()) {
			return nil, errors.New("epoch not found")
		}

		return nil, errors.Wrap(err, "failed to get epoch state")
	}

	return &epoch.Epoch, nil
}

func GetChains(ctx context.Context, chainType bridgetypes.ChainType, connection grpc.ClientConn) ([]bridgetypes.Chain, error) {
	chains, err := bridgetypes.NewQueryClient(connection).GetChainsByType(ctx, &bridgetypes.QueryGetChainsByType{
		ChainType: chainType,
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not get chains")
	}
	if len(chains.Chains) == 0 {
		return nil, errors.Errorf("no chains found")
	}

	return chains.Chains, nil
}

func GetParams(ctx context.Context, connection grpc.ClientConn) (*bridgetypes.Params, error) {
	params, err := bridgetypes.NewQueryClient(connection).Params(ctx, &bridgetypes.QueryParamsRequest{})
	if err != nil {
		return nil, errors.Wrap(err, "could not get params")
	}

	return &params.Params, nil
}
