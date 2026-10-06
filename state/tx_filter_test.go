package state_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbm "github.com/cometbft/cometbft-db"

	"github.com/cometbft/cometbft/crypto/ed25519"
	cmtrand "github.com/cometbft/cometbft/libs/rand"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/types"
)

func TestTxFilter(t *testing.T) {
	genDoc := randomGenesisDoc()
	genDoc.ConsensusParams.Block.MaxBytes = 6214
	genDoc.ConsensusParams.Evidence.MaxBytes = 1500

	// Max size of Txs is much smaller than size of block,
	// since we need to account for commits and evidence.
	testCases := []struct {
		tx    types.Tx
		isErr bool
	}{
		{types.Tx(cmtrand.Bytes(2122)), false},
		{types.Tx(cmtrand.Bytes(2123)), true},
		{types.Tx(cmtrand.Bytes(3000)), true},
	}

	for i, tc := range testCases {
		stateDB, err := dbm.NewDB("state", "memdb", os.TempDir())
		require.NoError(t, err)
		stateStore := sm.NewStore(stateDB, sm.StoreOptions{
			DiscardABCIResponses: false,
		})
		state, err := stateStore.LoadFromDBOrGenesisDoc(genDoc)
		require.NoError(t, err)

		f := sm.TxPreCheck(state)
		if tc.isErr {
			assert.NotNil(t, f(tc.tx), "#%v", i)
		} else {
			assert.Nil(t, f(tc.tx), "#%v", i)
		}
	}
}

func TestTxPreCheckSmallMaxBytes(t *testing.T) {
	state := stateFromGenesis(t)

	// Issue #6098: MaxBytes=1 used to panic inside MaxDataBytesNoEvidence.
	state.ConsensusParams.Block.MaxBytes = 1
	assertTxPreCheckTooSmall(t, state)

	// The one-validator floor is exactly enough for this genesis set, so the
	// filter is the normal size check rather than the undersized-params guard.
	require.Equal(t, 1, state.Validators.Size())
	state.ConsensusParams.Block.MaxBytes = types.MinBlockSizeBytes
	var filter func(types.Tx) error
	require.NotPanics(t, func() {
		filter = sm.TxPreCheck(state)
	})
	err := filter(types.Tx([]byte("test_tx")))
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "too small")

	// ValidateBasic accepts MinBlockSizeBytes, but two validators still do not fit.
	state.Validators = types.NewValidatorSet([]*types.Validator{
		types.NewValidator(ed25519.GenPrivKey().PubKey(), 10),
		types.NewValidator(ed25519.GenPrivKey().PubKey(), 10),
	})
	assertTxPreCheckTooSmall(t, state)

	state.ConsensusParams.Block.MaxBytes = types.MinBlockBytes(state.Validators.Size())
	require.NotPanics(t, func() {
		filter = sm.TxPreCheck(state)
	})
	err = filter(types.Tx([]byte("test_tx")))
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "too small")
}

func assertTxPreCheckTooSmall(t *testing.T, state sm.State) {
	t.Helper()
	var filter func(types.Tx) error
	require.NotPanics(t, func() {
		filter = sm.TxPreCheck(state)
	})
	err := filter(types.Tx([]byte("test_tx")))
	require.Error(t, err)
	assert.ErrorContains(t, err, "too small to accommodate block overhead and commit")
}

func stateFromGenesis(t *testing.T) sm.State {
	t.Helper()
	genDoc := randomGenesisDoc()
	stateDB, err := dbm.NewDB("state", "memdb", os.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, stateDB.Close())
	})
	stateStore := sm.NewStore(stateDB, sm.StoreOptions{
		DiscardABCIResponses: false,
	})
	state, err := stateStore.LoadFromDBOrGenesisDoc(genDoc)
	require.NoError(t, err)
	return state
}
