package main

import (
	"fmt"
	"strconv"

	"github.com/Harshjha3006/golang-blockchain/blockchain"
)

func main() {
	chain := blockchain.InitBlockChain()
	chain.AddBlock("1st block after Genesis")
	chain.AddBlock("2nd block after Genesis")
	chain.AddBlock("3rd block after Genesis")

	for _, block := range chain.Blocks {
		fmt.Printf("Data: %s\nPrevHash: %x\nHash: %x\n", block.Data, block.PrevHash, block.Hash)
		pow := blockchain.InitPow(block)
		fmt.Printf("POW : %s", strconv.FormatBool(pow.Validate()))
		fmt.Println()
	}

}
