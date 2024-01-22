package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"log"

	"golang.org/x/crypto/ripemd160"
)

const (
	checkSumLength = 4
	version        = byte(0x00)
)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

func NewKeyPair() (ecdsa.PrivateKey, []byte) {
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	pub := append(private.X.Bytes(), private.Y.Bytes()...)
	return *private, pub
}

func MakeWallet() *Wallet {
	private, pub := NewKeyPair()
	wallet := &Wallet{private, pub}
	return wallet
}

func (w Wallet) Address() []byte {
	// getting pubkeyHash
	pubHash := PubkeyHash(w.PublicKey)
	versionedHash := append([]byte{version}, pubHash...)
	checksum := CheckSum(versionedHash)

	fullHash := append(versionedHash, checksum...)
	address := Base58Encode(fullHash)
	return address

}
func PubkeyHash(pubkey []byte) []byte {
	sha256Hash := sha256.Sum256(pubkey)
	hasher := ripemd160.New()
	_, err := hasher.Write(sha256Hash[:])
	if err != nil {
		log.Panic(err)
	}
	pubkeyHash := hasher.Sum(nil)
	return pubkeyHash
}

func CheckSum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])
	return secondHash[:checkSumLength]
}
