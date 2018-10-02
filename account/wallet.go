package account

import (
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/pkg/errors"
	"sync"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/core/types"
)

type Wallet interface {
	Accounts() []*Account
	CreateAccount() (*Account, error)
	Contains(account *Account) bool
	SignHash(account *Account, hash []byte) ([]byte, error)
	SignTx(account *Account, tx *types.Transaction) (*types.Transaction, error)
}

type TinyWallet struct {
	mu       sync.RWMutex
	accounts []*Account
	unlocked map[common.Address]*Key
}

func NewTinyWallet(accounts []*Account) *TinyWallet {
	return &TinyWallet{
		accounts: accounts,
		unlocked: make(map[common.Address]*Key),
	}
}

func (tw *TinyWallet) Accounts() []*Account {
	tw.mu.RLock()
	defer tw.mu.RUnlock()
	var accounts []*Account
	for _, acc := range tw.accounts {
		accounts = append(accounts, acc)
	}
	return accounts
}

func (tw *TinyWallet) Contains(account *Account) bool {
	for _, acc := range tw.accounts {
		if acc.Address == account.Address {
			return true
		}
	}
	return false
}

func (tw *TinyWallet) CreateAccount() (*Account, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	acc, err := NewAccount()
	if err != nil {
		return nil, err
	}
	tw.accounts = append(tw.accounts, acc)
	return acc, nil
}

// Lock the account
func (tw *TinyWallet) Lock(address common.Address, priv crypto.PrivKey) error {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if validatePrivKey(address, priv) {
		return errors.New("Private key not match")
	}
	for addr, key := range tw.unlocked {
		if addr == address && key.privKey == priv {
			delete(tw.unlocked, addr)
			return nil
		}
	}
	return errors.New("This account has not been unlocked")
}

// Unlock the account, register key in wallet
func (tw *TinyWallet) Unlock(address common.Address, key crypto.PrivKey) error {
	tw.mu.RLock()
	if _, exist := tw.unlocked[address]; exist {
		return nil
	}
	tw.mu.RUnlock()
	acc, err := NewAccountWithKey(key)
	if err != nil {
		log.Error("Failed to create account")
		return ErrUnlockAcc
	}
	if acc.Address != address {
		log.Errorf("Address gen by private key is not equal to the target address")
		return ErrUnlockAcc
	}
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if !tw.Contains(acc) {
		tw.accounts = append(tw.accounts, acc)
	}
	tw.unlocked[address] = acc.key
	return nil
}

func (tw *TinyWallet) SignHash(account *Account, hash []byte) ([]byte, error) {
	if !tw.Contains(account) {
		return nil, ErrNotFoundAcc
	}
	key, ok := tw.unlocked[account.Address]
	if !ok {
		log.Error("account has not been unlocked")
		return nil, ErrNotUnlock
	}
	return key.privKey.Sign(hash)
}

func (tw *TinyWallet) SignTx(account *Account, tx *types.Transaction) ([]byte, error) {
	if !tw.Contains(account) {
		return nil, ErrNotFoundAcc
	}
	key, ok := tw.unlocked[account.Address]
	if !ok {
		return nil, ErrNotUnlock
	}
	return tx.Sign(key.privKey)
}
