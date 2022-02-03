package sync

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup/derive"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// SyncSource implements SyncReference with a L2 block sources and L1 hash-by-number source
type SyncSource struct {
	L1 interface {
		BlockLinkByNumber(ctx context.Context, num uint64) (self eth.BlockID, parent eth.BlockID, err error)
	}
	L2 interface {
		BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
		BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	}
}

// RefByL1Num fetches the canonical L1 block hash and the parent for the given L1 block height.
func (src SyncSource) RefByL1Num(ctx context.Context, l1Num uint64) (self eth.BlockID, parent eth.BlockID, err error) {
	return src.L1.BlockLinkByNumber(ctx, l1Num)
}

// RefByL2Num fetches the L1 and L2 block IDs from the engine for the given L2 block height.
// Use a nil height to fetch the head.
func (src SyncSource) RefByL2Num(ctx context.Context, l2Num *big.Int, genesis *rollup.Genesis) (refL1 eth.BlockID, refL2 eth.BlockID, parentL2 common.Hash, err error) {
	refL2Block, err2 := src.L2.BlockByNumber(ctx, l2Num) // nil for latest block
	if err2 != nil {
		err = fmt.Errorf("failed to retrieve L2 block: %v", err2)
		return
	}
	return derive.BlockReferences(refL2Block, genesis)
}

// RefByL2Hash fetches the L1 and L2 block IDs from the engine for the given L2 block hash.
func (src SyncSource) RefByL2Hash(ctx context.Context, l2Hash common.Hash, genesis *rollup.Genesis) (refL1 eth.BlockID, refL2 eth.BlockID, parentL2 common.Hash, err error) {
	refL2Block, err2 := src.L2.BlockByHash(ctx, l2Hash)
	if err2 != nil {
		err = fmt.Errorf("failed to retrieve L2 block: %v", err2)
		return
	}
	return derive.BlockReferences(refL2Block, genesis)
}

// SyncReference helps inform the sync algorithm of the L2 sync-state and L1 canonical chain
type SyncReference interface {
	// RefByL1Num fetches the canonical L1 block hash and the parent for the given L1 block height.
	RefByL1Num(ctx context.Context, l1Num uint64) (self eth.BlockID, parent eth.BlockID, err error)

	// RefByL2Num fetches the L1 and L2 block IDs from the engine for the given L2 block height.
	// Use a nil height to fetch the head.
	RefByL2Num(ctx context.Context, l2Num *big.Int, genesis *rollup.Genesis) (refL1 eth.BlockID, refL2 eth.BlockID, parentL2 common.Hash, err error)

	// RefByL2Hash fetches the L1 and L2 block IDs from the engine for the given L2 block hash.
	RefByL2Hash(ctx context.Context, l2Hash common.Hash, genesis *rollup.Genesis) (refL1 eth.BlockID, refL2 eth.BlockID, parentL2 common.Hash, err error)
}
