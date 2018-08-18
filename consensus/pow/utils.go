package pow

import (
	"tinychain/common"
	"tinychain/core/types"
	"math/big"
	"encoding/binary"
)

func computeHash(nonce uint64, header *types.Header) ([]byte, error) {
	var nonceBytes []byte
	binary.BigEndian.PutUint64(nonceBytes, nonce)
	hash := header.Hash().Bytes()
	hash = append(hash, nonceBytes...)
	return common.Sha256(hash), nil
}

func computeTarget(difficulty uint64) *big.Int {
	target := new(big.Int).SetUint64(1)
	return target.Lsh(target, uint(256-difficulty))
}
