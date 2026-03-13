package helpers

import (
	"context"

	bridgetypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/cosmos/gogoproto/grpc"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

func GetEpochState(ctx context.Context, epochID uint32, connection grpc.ClientConn) (*bridgetypes.Epoch, error) {
	epoch, err := bridgetypes.NewQueryClient(connection).GetEpochById(ctx, &bridgetypes.QueryGetEpoch{
		EpochId: epochID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not get epoch")
	}

	if epoch == nil {
		return nil, errors.New("epoch not found")
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
