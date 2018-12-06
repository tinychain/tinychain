package crypto

import "github.com/libp2p/go-libp2p-crypto"

type PublicKey interface {
	crypto.PubKey

	// VerifyVRF asserts that proof is correct for m and outputs index.
	VerifyVRF(m, proof, value []byte) error
}

type PrivateKey interface {
	crypto.PrivKey

	// Evaluate returns the verifiable unpredictable function evaluated at m.
	Evaluate(m []byte) (value, proof []byte)
}

func NewKeyPair() (PrivateKey, PublicKey, error) {

}
