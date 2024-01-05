package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
)

type Block struct {
	Hash     []byte
	Data     []byte
	PrevHash []byte
}

func (b *Block) DeriveHash() {
	temp := bytes.Join([][]byte{b.Data, b.PrevHash}, []byte{})
	hash := sha256.Sum256(temp)
	b.Hash = hash[:]
}

func CreateBlock(data []byte, prevHash []byte) (b *Block) {
	newBlock := &Block{[]byte{}, data, prevHash}
	newBlock.DeriveHash()
	return newBlock
}

func Genesis() (b *Block) {
	newBlock := CreateBlock([]byte("Genesis"), []byte{})
	return newBlock
}

type BlockChain struct {
	blocks []*Block
}

func (chain *BlockChain) AddBlock(data string) {
	lastBlock := chain.blocks[len(chain.blocks)-1]
	newBlock := CreateBlock([]byte(data), lastBlock.Hash)
	chain.blocks = append(chain.blocks, newBlock)
}

func InitBlockChain() *BlockChain {
	newChain := &BlockChain{[]*Block{Genesis()}}
	return newChain
}

func main() {
	chain := InitBlockChain()
	chain.AddBlock("1st block after Genesis")
	chain.AddBlock("2nd block after Genesis")
	chain.AddBlock("3rd block after Genesis")

	for _, block := range chain.blocks {
		fmt.Printf("Data: %s\nPrevHash: %x\nHash: %x\n", block.Data, block.PrevHash, block.Hash)
	}

}
