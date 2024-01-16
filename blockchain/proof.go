package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/big"
)

type ProofOfWork struct {
	Block  *Block
	Target *big.Int
}

// take data from the block

const Difficulty = 12

func InitPow(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-Difficulty))
	pow := &ProofOfWork{b, target}
	return pow
}

func (p *ProofOfWork) Validate() bool {
	var intHash big.Int
	var hash [32]byte

	data := p.InitData(p.Block.Nonce)
	hash = sha256.Sum256(data)
	intHash.SetBytes(hash[:])
	return intHash.Cmp(p.Target) == -1
}
func (p *ProofOfWork) Run() (int, []byte) {
	var intHash big.Int
	var hash [32]byte

	nonce := 0
	for nonce < math.MaxInt64 {
		data := p.InitData(nonce)
		hash = sha256.Sum256(data)
		fmt.Printf("\r%x", hash)

		intHash.SetBytes(hash[:])

		if intHash.Cmp(p.Target) == -1 {
			break
		} else {
			nonce++
		}
	}
	fmt.Println()
	return nonce, hash[:]
}

func (p *ProofOfWork) InitData(nonce int) []byte {
	res := bytes.Join([][]byte{p.Block.HashTransactions(), p.Block.PrevHash, ToHex(int64(nonce)), ToHex(int64(Difficulty))}, []byte{})
	return res
}

func ToHex(num int64) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, num)
	if err != nil {
		log.Panic()
	}
	return buf.Bytes()
}

// initiate nonce at 0

// compute hash of nonce,prevHash,difficulty and data

// increment nonce until hash meets requirements

// requirements : first few bits of the hash should be zero
