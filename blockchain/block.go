package blockchain

import (
	"bytes"
	"encoding/gob"
	"log"
	"time"
)

type Block struct {
	Timstamp     int64
	Hash         []byte
	Transactions []*Transaction
	PrevHash     []byte
	Height       int
	Nonce        int
}

func CreateBlock(txs []*Transaction, prevHash []byte, height int) (b *Block) {
	newBlock := &Block{time.Now().Unix(), []byte{}, txs, prevHash, height, 0}
	pow := InitPow(newBlock)

	nonce, hash := pow.Run()
	newBlock.Nonce = nonce
	newBlock.Hash = hash[:]

	return newBlock
}

func Genesis(coinbase *Transaction) (b *Block) {
	newBlock := CreateBlock([]*Transaction{coinbase}, []byte{}, 0)
	return newBlock
}

func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte

	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.Serialize())
	}
	tree := NewMerkleTree(txHashes)
	return tree.RootNode.Data
}
func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)
	err := encoder.Encode(b)

	Handle(err)
	return res.Bytes()

}

func Deserialize(data []byte) *Block {
	decoder := gob.NewDecoder(bytes.NewReader(data))
	var res Block
	err := decoder.Decode(&res)
	Handle(err)
	return &res
}

func Handle(err error) {
	if err != nil {
		log.Panic(err)
	}
}
