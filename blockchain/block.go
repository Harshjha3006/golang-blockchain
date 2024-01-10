package blockchain

import (
	"bytes"
	"encoding/gob"
	"log"
)

type Block struct {
	Hash     []byte
	Data     []byte
	PrevHash []byte
	Nonce    int
}

func CreateBlock(data []byte, prevHash []byte) (b *Block) {
	newBlock := &Block{[]byte{}, data, prevHash, 0}
	pow := InitPow(newBlock)

	nonce, hash := pow.Run()
	newBlock.Nonce = nonce
	newBlock.Hash = hash[:]

	return newBlock
}

func Genesis() (b *Block) {
	newBlock := CreateBlock([]byte("Genesis"), []byte{})
	return newBlock
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
