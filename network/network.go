package network

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"runtime"
	"syscall"

	"github.com/Harshjha3006/golang-blockchain/blockchain"
	"github.com/vrecan/death/v3"
)

const (
	protocol   = "tcp"
	commandLen = 12
	version    = 1
)

var (
	KnownNodes      = []string{"localhost:3000"} // the address of the full node
	nodeAddress     string
	blocksInTransit = [][]byte{}
	memoryPool      = make(map[string]blockchain.Transaction)
	minerAddress    string
)

type Addr struct {
	AddrList []string
}
type Block struct {
	AddrFrom string
	Block    []byte
}

type Transaction struct {
	AddrFrom    string
	Transaction []byte
}

type GetBlocks struct {
	AddrFrom string
}

type GetData struct {
	AddrFrom string
	Kind     string
	Id       []byte
}
type Inv struct {
	AddrFrom string
	Kind     string
	Items    [][]byte
}

type Version struct {
	Version     int
	BestHeight  int
	NodeAddress string
}

func RequestBlocks() {
	for _, node := range KnownNodes {
		SendGetBlocks(node)
	}
}
func SendBlock(addr string, b *blockchain.Block) {
	data := Block{nodeAddress, b.Serialize()}
	payload := GobEncode(data)
	payload = append(CmdToBytes("block"), payload...)
	SendData(addr, payload)

}

func SendAddr(addr string) {
	var nodes = Addr{KnownNodes}
	nodes.AddrList = append(nodes.AddrList, nodeAddress)
	payload := GobEncode(nodes)
	payload = append(CmdToBytes("addr"), payload...)
	SendData(addr, payload)
}
func SendTransaction(addr string, tx *blockchain.Transaction) {
	data := Transaction{nodeAddress, tx.Serialize()}
	payload := GobEncode(data)
	payload = append(CmdToBytes("tx"), payload...)
	SendData(addr, payload)
}

func SendGetBlocks(addr string) {
	payload := GobEncode(GetBlocks{nodeAddress})
	payload = append(CmdToBytes("getblocks"), payload...)
	SendData(addr, payload)
}

func SendGetData(addr string, Kind string, id []byte) {
	getData := GetData{nodeAddress, Kind, id}
	payload := GobEncode(getData)
	payload = append(CmdToBytes("getdata"), payload...)
	SendData(addr, payload)
}

func SendVersion(addr string, chain *blockchain.BlockChain) {
	bestHeight := chain.GetBestHeight()
	payload := GobEncode(Version{Version: version, BestHeight: bestHeight, NodeAddress: nodeAddress})
	payload = append(CmdToBytes("version"), payload...)
	SendData(addr, payload)

}

func SendInv(addr string, Kind string, items [][]byte) {
	inv := Inv{nodeAddress, Kind, items}
	payload := GobEncode(inv)
	payload = append(CmdToBytes("inv"), payload...)
	SendData(addr, payload)
}
func SendData(addr string, data []byte) {

	conn, err := net.Dial(protocol, addr)

	if err != nil {
		fmt.Printf("%s is not available\n", addr)

		// var updatedNodes []string

		// for _, node := range KnownNodes {
		// 	if node != addr {
		// 		updatedNodes = append(updatedNodes, node)
		// 	}
		// }
		// KnownNodes = updatedNodes
		return
	}
	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))

	if err != nil {
		log.Panic(err)
	}

}
func HandleAddr(request []byte) {
	var buffer bytes.Buffer
	var payload Addr

	buffer.Write(request[commandLen:])

	decoder := gob.NewDecoder(&buffer)
	err := decoder.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	KnownNodes = append(KnownNodes, payload.AddrList...)
	fmt.Printf("There are %d known nodes", len(KnownNodes))
	RequestBlocks()
}
func HandleInv(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Inv

	buff.Write(request[commandLen:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("Recevied inventory with %d %s from %s\n", len(payload.Items), payload.Kind, payload.AddrFrom)

	if payload.Kind == "block" {
		blocksInTransit = payload.Items

		blockHash := payload.Items[0]
		SendGetData(payload.AddrFrom, "block", blockHash)

		newInTransit := [][]byte{}
		for _, b := range blocksInTransit {
			if !bytes.Equal(b, blockHash) {
				newInTransit = append(newInTransit, b)
			}
		}
		blocksInTransit = newInTransit
	}

	if payload.Kind == "tx" {
		txID := payload.Items[0]

		if memoryPool[hex.EncodeToString(txID)].Id == nil {
			SendGetData(payload.AddrFrom, "tx", txID)
		}
	}
}
func HandleVersion(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Version

	buff.Write(request[commandLen:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	bestHeight := chain.GetBestHeight()
	otherHeight := payload.BestHeight

	if bestHeight < otherHeight {
		SendGetBlocks(payload.NodeAddress)
	} else if bestHeight > otherHeight {
		SendVersion(payload.NodeAddress, chain)
	}

	if !NodeIsKnown(payload.NodeAddress) {
		KnownNodes = append(KnownNodes, payload.NodeAddress)
	}
}
func NodeIsKnown(addr string) bool {
	for _, node := range KnownNodes {
		if node == addr {
			return true
		}
	}
	return false
}

func HandleGetBlocks(request []byte, chain *blockchain.BlockChain) {
	var buffer bytes.Buffer
	var payload GetBlocks

	buffer.Write(request[commandLen:])

	decoder := gob.NewDecoder(&buffer)
	err := decoder.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}
	blocks := chain.GetBlockHashes()
	SendInv(payload.AddrFrom, "block", blocks)
}
func HandleBlock(request []byte, chain *blockchain.BlockChain) {
	var buffer bytes.Buffer
	var payload Block

	buffer.Write(request[commandLen:])

	decoder := gob.NewDecoder(&buffer)
	err := decoder.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}
	blockData := payload.Block
	block := blockchain.Deserialize(blockData)

	fmt.Printf("Received a new block")
	chain.AddBlock(block)
	fmt.Printf("Added block %x\n", block.Hash)

	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		SendGetData(payload.AddrFrom, "block", blockHash)
		blocksInTransit = blocksInTransit[1:]
	} else {
		utxoset := blockchain.UTXOSet{Blockchain: chain}
		utxoset.ReIndex()
	}
}
func HandleGetData(request []byte, chain *blockchain.BlockChain) {
	var buffer bytes.Buffer
	var payload GetData

	buffer.Write(request[commandLen:])

	decoder := gob.NewDecoder(&buffer)
	err := decoder.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	if payload.Kind == "block" {
		block, err := chain.GetBlock([]byte(payload.Id))
		if err != nil {
			return
		}
		SendBlock(payload.AddrFrom, &block)
	} else if payload.Kind == "tx" {
		txId := hex.EncodeToString(payload.Id)
		tx := memoryPool[txId]
		SendTransaction(payload.AddrFrom, &tx)
	}
}

func HandleTransaction(req []byte, chain *blockchain.BlockChain) {
	var buffer bytes.Buffer
	var payload Transaction

	buffer.Write(req[commandLen:])

	decoder := gob.NewDecoder(&buffer)
	err := decoder.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}
	txData := payload.Transaction
	tx := blockchain.DeserializeTransaction(txData)
	memoryPool[hex.EncodeToString(tx.Id)] = tx

	fmt.Printf("%s, %d\n", nodeAddress, len(memoryPool))

	if nodeAddress == KnownNodes[0] {
		for _, node := range KnownNodes {
			if node != KnownNodes[0] && node != payload.AddrFrom {
				SendInv(node, "tx", [][]byte{tx.Id})
			}
		}
	} else {
		if len(memoryPool) >= 2 && len(minerAddress) > 0 {
			MineTx(chain)
		}
	}

}

func MineTx(chain *blockchain.BlockChain) {
	var txs []*blockchain.Transaction

	for id := range memoryPool {
		fmt.Printf("tx : %s\n", memoryPool[id].Id)
		tx := memoryPool[id]
		if chain.VerifyTransaction(&tx) {
			txs = append(txs, &tx)
		}
	}
	if len(txs) <= 0 {
		fmt.Printf("All transactions are invalid")
		return
	}

	cbtx := blockchain.CoinbaseTx(minerAddress, "")
	txs = append(txs, cbtx)
	newBlock := chain.MineBlock(txs)
	utxoset := blockchain.UTXOSet{Blockchain: chain}
	utxoset.ReIndex()

	fmt.Println("New Block added")
	for _, tx := range txs {
		txID := hex.EncodeToString(tx.Id)
		delete(memoryPool, txID)
	}

	for _, node := range KnownNodes {
		if node != nodeAddress {
			SendInv(node, "block", [][]byte{newBlock.Hash})
		}
	}

	if len(memoryPool) > 0 {
		MineTx(chain)
	}
}
func GobEncode(data interface{}) []byte {
	var buf bytes.Buffer

	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(data)
	if err != nil {
		log.Panic(err)
	}
	return buf.Bytes()
}

func CmdToBytes(cmd string) []byte {
	var bytes [commandLen]byte

	for i, c := range cmd {
		bytes[i] = byte(c)
	}
	return bytes[:]
}
func BytesToCmd(bytes []byte) string {
	var cmd []byte

	for _, b := range bytes {
		if b != 0x0 {
			cmd = append(cmd, b)
		}
	}
	return string(cmd)
}

func HandleConnection(conn net.Conn, chain *blockchain.BlockChain) {
	req, err := ioutil.ReadAll(conn)
	defer conn.Close()
	if err != nil {
		log.Panic(err)
	}
	command := BytesToCmd(req[:commandLen])

	fmt.Printf("Received %s command\n", command)
	switch command {
	case "addr":
		HandleAddr(req)
	case "block":
		HandleBlock(req, chain)
	case "inv":
		HandleInv(req, chain)
	case "getblocks":
		HandleGetBlocks(req, chain)
	case "getdata":
		HandleGetData(req, chain)
	case "version":
		HandleVersion(req, chain)
	case "tx":
		HandleTransaction(req, chain)
	default:
		fmt.Printf("Unknown Command")
	}
}
func StartServer(nodeId string, mineAddress string) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeId)
	minerAddress = mineAddress
	fmt.Println()
	ln, err := net.Listen(protocol, nodeAddress)

	if err != nil {
		log.Panic(err)
	}

	defer ln.Close()
	chain := blockchain.ContinueBlockChain(nodeId)
	defer chain.Database.Close()
	go CloseDb(chain)

	if nodeAddress != KnownNodes[0] {
		SendVersion(KnownNodes[0], chain)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}
		go HandleConnection(conn, chain)
	}
}

func CloseDb(chain *blockchain.BlockChain) {
	d := death.NewDeath(syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	d.WaitForDeathWithFunc(func() {
		defer os.Exit(1)
		defer runtime.Goexit()
		chain.Database.Close()
	})
}
