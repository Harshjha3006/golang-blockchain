package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"

	"github.com/Harshjha3006/golang-blockchain/blockchain"
	"github.com/Harshjha3006/golang-blockchain/network"
	"github.com/Harshjha3006/golang-blockchain/wallet"
)

type Cmd struct{}

func (cli *Cmd) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("getbalance -address ADDRESS - prints the balance of the specified address")
	fmt.Println("createblockchain - address ADDRESS - creats a new blockchain and sends genesis reward to specified address")
	fmt.Println("printchain - prints the entire blockchain")
	fmt.Println(" send -from FROM -to TO -amount AMOUNT -mine - Send amount of coins. Then -mine flag is set, mine off of this node")
	fmt.Println("createwallet - Creates a New Wallet")
	fmt.Println("listaddress - Lists all addresses in your wallet")
	fmt.Println("reindexutxo - Reindexes your utxo database")
	fmt.Println(" startnode -miner ADDRESS - Start a node with ID specified in NODE_ID env. var. -miner enables mining")
}
func (cli *Cmd) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *Cmd) startNode(nodeId, minerAddress string) {
	fmt.Printf("Starting Node %s\n", nodeId)
	if len(minerAddress) > 0 {
		if wallet.ValidateAddress(minerAddress) {
			fmt.Printf("Mining is on, rewards will be received at %s", minerAddress)
		} else {
			log.Panic("Invalid address")
		}
	}
	network.StartServer(nodeId, minerAddress)
}
func (cli *Cmd) reindex(nodeId string) {
	chain := blockchain.ContinueBlockChain(nodeId)
	defer chain.Database.Close()
	Utxoset := blockchain.UTXOSet{Blockchain: chain}
	Utxoset.ReIndex()
	count := Utxoset.CountTransactions()
	fmt.Printf("There are %v transactions in the UTXO set \n", count)
}

func (cli *Cmd) printChain(nodeId string) {
	chain := blockchain.ContinueBlockChain(nodeId)
	defer chain.Database.Close()
	iter := chain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("PrevHash: %x\nHash: %x\n", block.PrevHash, block.Hash)
		pow := blockchain.InitPow(block)
		fmt.Printf("POW : %s", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Println()
		if len(block.PrevHash) == 0 {
			break
		}
	}
}
func (cli *Cmd) createBlockChain(address string, nodeId string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not valid")
	}
	chain := blockchain.InitBlockChain(address, nodeId)
	defer chain.Database.Close()

	UtxoSet := blockchain.UTXOSet{Blockchain: chain}
	UtxoSet.ReIndex()
	fmt.Println("Finished")
}

func (cli *Cmd) getBalance(address string, nodeId string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not valid")
	}
	chain := blockchain.ContinueBlockChain(nodeId)
	defer chain.Database.Close()
	pubKeyHash := wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]

	utxoSet := blockchain.UTXOSet{Blockchain: chain}
	utxos := utxoSet.FindUTXO(pubKeyHash)

	balance := 0
	for _, out := range utxos {
		balance += out.Value
	}
	fmt.Printf("The balance of %s is %d\n", address, balance)
}

func (cli *Cmd) send(from string, to string, amount int, nodeId string, mine bool) {
	if !wallet.ValidateAddress(from) {
		log.Panic("Address is not valid")
	}
	if !wallet.ValidateAddress(to) {
		log.Panic("Address is not valid")
	}
	chain := blockchain.ContinueBlockChain(nodeId)
	defer chain.Database.Close()
	utxoSet := blockchain.UTXOSet{Blockchain: chain}
	wallets, err := wallet.CreateWallets(nodeId)
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)
	txn := blockchain.NewTransaction(&wallet, to, amount, utxoSet)

	if mine {
		cbtx := blockchain.CoinbaseTx(from, "")
		txns := []*blockchain.Transaction{cbtx, txn}
		block := chain.MineBlock(txns)
		utxoSet.Update(block)

	} else {
		network.SendTransaction(network.KnownNodes[0], txn)
		fmt.Println("Transaction sent")
	}
	fmt.Println("success")
}

func (cli *Cmd) createWallet(nodeId string) {
	wallets, _ := wallet.CreateWallets(nodeId)
	address := wallets.AddWallet()
	wallets.SaveFile(nodeId)
	fmt.Printf("The new Address is %s\n", address)
}
func (cli *Cmd) listAddress(nodeId string) {
	wallets, _ := wallet.CreateWallets(nodeId)
	addresses := wallets.GetAllAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}

}
func (cli *Cmd) Run() {
	cli.validateArgs()

	nodeId := os.Getenv("NODE_ID")
	if nodeId == "" {
		fmt.Printf("nodeId env variable not set")
		runtime.Goexit()
	}

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressCmd := flag.NewFlagSet("listaddress", flag.ExitOnError)
	reindexUtxo := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	sendMine := sendCmd.Bool("mine", false, "Mine immediately on the same node")
	startNodeMiner := startNodeCmd.String("miner", "", "Enable miner and you can mine blocks and send reward to Address")

	switch os.Args[1] {
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "listaddress":
		err := listAddressCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
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
	case "reindexutxo":
		err := reindexUtxo.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if reindexUtxo.Parsed() {
		cli.reindex(nodeId)
	}
	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.getBalance(*getBalanceAddress, nodeId)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockChain(*createBlockchainAddress, nodeId)
	}

	if printChainCmd.Parsed() {
		cli.printChain(nodeId)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}

		cli.send(*sendFrom, *sendTo, *sendAmount, nodeId, *sendMine)
	}
	if createWalletCmd.Parsed() {
		cli.createWallet(nodeId)
	}
	if listAddressCmd.Parsed() {
		cli.listAddress(nodeId)
	}
	if startNodeCmd.Parsed() {
		cli.startNode(nodeId, *startNodeMiner)
	}
}
