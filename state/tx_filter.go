package state

import (
	"fmt"

	mempl "github.com/cometbft/cometbft/mempool"
	"github.com/cometbft/cometbft/types"
)

// TxPreCheck returns a function to filter transactions before processing.
// The function limits the size of a transaction to the block's maximum data size.
//
// If Block.MaxBytes cannot fit the header and commit for the current validator
// set, the returned filter rejects every transaction instead of panicking.
// Node startup (node/setup.go) and the post-commit mempool refresh both call
// TxPreCheck on whatever is already committed, so a value that slipped past
// validation must not crash the process. Proposal creation still panics in
// MaxDataBytes when the same value cannot fit a block; that halt is intentional.
func TxPreCheck(state State) mempl.PreCheckFunc {
	maxBytes := state.ConsensusParams.Block.MaxBytes
	if maxBytes == -1 {
		maxBytes = int64(types.MaxBlockSizeBytes)
	}
	valsCount := state.Validators.Size()
	minRequired := types.MinBlockBytes(valsCount)
	if maxBytes < minRequired {
		return func(tx types.Tx) error {
			return fmt.Errorf("block.MaxBytes (%d) is too small to accommodate block overhead and commit (%d)", maxBytes, minRequired)
		}
	}
	maxDataBytes := types.MaxDataBytesNoEvidence(
		maxBytes,
		valsCount,
	)
	return mempl.PreCheckMaxBytes(maxDataBytes)
}

// TxPostCheck returns a function to filter transactions after processing.
// The function limits the gas wanted by a transaction to the block's maximum total gas.
func TxPostCheck(state State) mempl.PostCheckFunc {
	return mempl.PostCheckMaxGas(state.ConsensusParams.Block.MaxGas)
}
