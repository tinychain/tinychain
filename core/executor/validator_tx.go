package executor

import (
	"tinychain/core/types"
	"tinychain/core/state"
	"errors"
)

var (
	ErrTxTooLarge    = errors.New("oversized data")
	ErrNegativeValue = errors.New("negative value")
	ErrGasLimit      = errors.New("exceeds block gas limit")
	ErrInvalidSender = errors.New("invalid sender")
)

type TxValidatorImpl struct {
	config *Config
	state  *state.StateDB
}

func NewTxValidator(config *Config, state *state.StateDB) TxValidator {
	return &TxValidatorImpl{
		config: config,
		state:  state,
	}
}

func (v *TxValidatorImpl) ValidateTxs(txs types.Transactions) (valid types.Transactions, invalid types.Transactions) {
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
// 3. check tx gas exceed the current block gas limit or not
// 4. check address format is valid or not
// 5. check signature
// 6. check nonce
// 7. check balance is enough or not for tx.Cost()
func (v *TxValidatorImpl) ValidateTx(tx *types.Transaction) error {
	if tx.Size() > types.MaxTxSize {
		return ErrTxTooLarge
	}

	if tx.Value.Sign() < 0 {
		return ErrNegativeValue
	}
	// TODO tx validates

	return nil
}
