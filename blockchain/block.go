package blockchain

import (
	"bytes"
	"encoding/gob"
	"log"
)

type Block struct {
	Hash         []byte
	Transactions []*Transaction
	PrevHash     []byte
	Nonce        int
}

func CreateBlock(txs []*Transaction, prevHash []byte) (b *Block) {
	newBlock := &Block{[]byte{}, txs, prevHash, 0}
	pow := InitPow(newBlock)

	nonce, hash := pow.Run()
	newBlock.Nonce = nonce
	newBlock.Hash = hash[:]

	return newBlock
}

func Genesis(coinbase *Transaction) (b *Block) {
	newBlock := CreateBlock([]*Transaction{coinbase}, []byte{})
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
