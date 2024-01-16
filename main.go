package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"

	"github.com/Harshjha3006/golang-blockchain/blockchain"
)

type Cmd struct{}

func (cli *Cmd) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("getbalance -address ADDRESS - prints the balance of the specified address")
	fmt.Println("createblockchain - address ADDRESS - creats a new blockchain and sends genesis reward to specified address")
	fmt.Println("printchain - prints the entire blockchain")
	fmt.Println("send -from FROM -to TO -amount AMOUNT - sends a transaction")
}
func (cli *Cmd) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *Cmd) printChain() {
	chain := blockchain.ContinueBlockChain()
	defer chain.Database.Close()
	iter := chain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("PrevHash: %x\nHash: %x\n", block.PrevHash, block.Hash)
		pow := blockchain.InitPow(block)
		fmt.Printf("POW : %s", strconv.FormatBool(pow.Validate()))
		fmt.Println()
		fmt.Println()
		if len(block.PrevHash) == 0 {
			break
		}
	}
}
func (cli *Cmd) createBlockChain(address string) {
	chain := blockchain.InitBlockChain(address)
	chain.Database.Close()
	fmt.Println("Finished")
}

func (cli *Cmd) getBalance(address string) {
	chain := blockchain.ContinueBlockChain()
	defer chain.Database.Close()
	utxos := chain.FindUTXO(address)
	balance := 0
	for _, out := range utxos {
		balance += out.Value
	}
	fmt.Printf("The balance of %s is %d\n", address, balance)
}

func (cli *Cmd) send(from string, to string, amount int) {
	chain := blockchain.ContinueBlockChain()
	defer chain.Database.Close()
	tx := blockchain.NewTransaction(from, to, amount, chain)
	chain.AddBlock([]*blockchain.Transaction{tx})
	fmt.Println("Transaction Success")
}

func (cli *Cmd) run() {
	cli.validateArgs()

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")

	switch os.Args[1] {
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.getBalance(*getBalanceAddress)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockChain(*createBlockchainAddress)
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}

		cli.send(*sendFrom, *sendTo, *sendAmount)
	}
}
func main() {
	defer os.Exit(0)
	cli := Cmd{}

	cli.run()

}
