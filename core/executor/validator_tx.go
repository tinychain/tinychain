package executor

import (
	"errors"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/core/state"
	"github.com/tinychain/tinychain/core/types"

	"github.com/libp2p/go-libp2p-crypto"
)

var (
	errTxTooLarge    = errors.New("oversized data")
	errNegativeValue = errors.New("negative value")
	errSignMismatch  = errors.New("tx signature mismatch")
	errNonceTooLow   = errors.New("tx nonce too low")
	errInvalidSender = errors.New("invalid sender")
)

type TxValidator struct {
	config *common.Config
	state  *state.StateDB
}

func NewTxValidator(config *common.Config, state *state.StateDB) *TxValidator {
	return &TxValidator{
		config: config,
		state:  state,
	}
}

func (v *TxValidator) ValidateTxs(txs types.Transactions) (valid types.Transactions, invalid types.Transactions) {
	for _, tx := range txs {
		if err := v.ValidateTx(tx); err != nil {
			invalid = append(invalid, tx)
		} else {
			valid = append(valid, tx)
		}
	}
	return valid, invalid
}

// Validate transaction
// 1. check tx size
// 2. check tx value
// 3. check address is match
// 4. check signature is match
// 5. check nonce
// 6. check balance is enough or not for tx.Cost()
func (v *TxValidator) ValidateTx(tx *types.Transaction) error {
	// Check tx size
	if tx.Size() > types.MaxTxSize {
		return errTxTooLarge
	}

	// Check tx value
	if tx.Value.Sign() < 0 {
		return errNegativeValue
	}

	// Decode public key
	pubkey, err := crypto.UnmarshalPublicKey(tx.PubKey)
	if err != nil {
		return err
	}

	// Generate address
	addr, err := common.GenAddrByPubkey(pubkey)
	if err != nil {
		return err
	}

	// Check address is match tx.From or not
	if addr != tx.From {
		return errInvalidSender
	}

	// Verify signature
	match, err := pubkey.Verify(tx.Hash().Bytes(), tx.Signature)
	if err != nil {
		return err
	}
	if !match {
		return errSignMismatch
	}

	// Check nonce
	if nonce := v.state.GetNonce(tx.From); tx.Nonce < nonce {
		return errNonceTooLow
	}

	// Check balance
	if balance := v.state.GetBalance(tx.From); balance.Cmp(tx.Cost()) < 0 {
		return errBalanceNotEnough
	}

	return nil
}
