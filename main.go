package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/Harshjha3006/golang-blockchain/blockchain"
)

type Cmd struct {
	chain *blockchain.BlockChain
}

func (cli *Cmd) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("add -block BLOCK_DATA - adds a new block")
	fmt.Println("print - prints the entire blockchain")
}
func (cli *Cmd) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *Cmd) addBlock(data string) {
	cli.chain.AddBlock(data)
	fmt.Println("Block Added")
}
func (cli *Cmd) printChain() {
	iter := cli.chain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("Data: %s\nPrevHash: %x\nHash: %x\n", block.Data, block.PrevHash, block.Hash)
		pow := blockchain.InitPow(block)
		fmt.Printf("POW : %s", strconv.FormatBool(pow.Validate()))
		fmt.Println()
		fmt.Println()
		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *Cmd) run() {
	cli.validateArgs()

	addBlockCmd := flag.NewFlagSet("add", flag.ExitOnError)
	printBlockCmd := flag.NewFlagSet("print", flag.ExitOnError)
	addBlockData := addBlockCmd.String("block", "", "adds new block for given data")

	switch os.Args[1] {
	case "add":
		err := addBlockCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "print":
		err := printBlockCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if addBlockCmd.Parsed() {
		if *addBlockData == "" {
			addBlockCmd.Usage()
			runtime.Goexit()
		}
		cli.addBlock(*addBlockData)
	}
	if printBlockCmd.Parsed() {
		cli.printChain()
	}
}
func main() {
	defer os.Exit(0)
	chain := blockchain.InitBlockChain()
	defer chain.Database.Close()

	cli := Cmd{chain}

	cli.run()

}
