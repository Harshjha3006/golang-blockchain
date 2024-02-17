package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dgraph-io/badger"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

const (
	dbPath      = "./tmp/blocks_%s"
	genesisData = "First Transaction from Genesis"
)

func (chain *BlockChain) AddBlock(block *Block) {
	err := chain.Database.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(block.Hash); err == nil {
			return nil
		}

		blockData := block.Serialize()
		err := txn.Set(block.Hash, blockData)
		Handle(err)

		item, err := txn.Get([]byte("lh"))
		Handle(err)
		lastHash, err := item.Value()
		Handle(err)

		item, err = txn.Get(lastHash)
		Handle(err)
		lastBlockData, err := item.Value()
		Handle(err)

		lastBlock := *Deserialize(lastBlockData)

		if block.Height > lastBlock.Height {
			err = txn.Set([]byte("lh"), block.Hash)
			Handle(err)
			chain.LastHash = block.Hash
		}
		return nil
	})
	Handle(err)
}
func (chain *BlockChain) MineBlock(txs []*Transaction) *Block {
	var lastHash []byte
	var lastHeight int
	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		lastHash, err = item.Value()
		Handle(err)
		item, err = txn.Get(lastHash)
		Handle(err)
		blockData, err := item.Value()
		block := *Deserialize(blockData)
		lastHeight = block.Height
		return err
	})
	Handle(err)
	newBlock := CreateBlock(txs, lastHash, lastHeight+1)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err = txn.Set([]byte("lh"), newBlock.Hash)
		Handle(err)
		err = txn.Set(newBlock.Hash, newBlock.Serialize())
		return err
	})
	Handle(err)
	return newBlock
}

func (chain *BlockChain) GetBlock(blockHash []byte) (Block, error) {
	var block Block
	err := chain.Database.View(func(txn *badger.Txn) error {
		if item, err := txn.Get(blockHash); err != nil {
			return errors.New("Block not found")
		} else {
			blockData, _ := item.Value()
			block = *Deserialize(blockData)
		}
		return nil
	})
	if err != nil {
		return block, err
	}
	return block, nil
}

func (chain *BlockChain) GetBlockHashes() [][]byte {
	iter := chain.Iterator()
	var blocks [][]byte

	for {
		block := iter.Next()
		blocks = append(blocks, block.Hash)
		if len(block.PrevHash) == 0 {
			break
		}
	}
	return blocks
}

func (chain *BlockChain) GetBestHeight() int {
	var block Block
	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		lastHash, err := item.Value()
		Handle(err)
		item, err = txn.Get(lastHash)
		Handle(err)
		blockData, err := item.Value()
		block = *Deserialize(blockData)
		return err
	})
	Handle(err)
	return block.Height
}

func DbExists(path string) bool {
	if _, err := os.Stat(path + "/MANIFEST"); os.IsNotExist(err) {
		return false
	}
	return true
}

func ContinueBlockChain(nodeId string) *BlockChain {

	path := fmt.Sprintf(dbPath, nodeId)
	if !DbExists(path) {
		fmt.Println("No BlockChain found, Create one")
		runtime.Goexit()
	}

	var lastHash []byte
	opts := badger.DefaultOptions
	opts.Dir = path
	opts.ValueDir = path

	db, err := openDb(path, opts)
	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		lastHash, err = item.Value()
		return err
	})
	Handle(err)
	return &BlockChain{lastHash, db}
}

func InitBlockChain(address string, nodeId string) *BlockChain {

	path := fmt.Sprintf(dbPath, nodeId)
	if DbExists(path) {
		fmt.Println("BlockChain already exists")
		runtime.Goexit()
	}

	var lastHash []byte
	opts := badger.DefaultOptions
	opts.Dir = path
	opts.ValueDir = path

	db, err := openDb(path, opts)
	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		coinbase := CoinbaseTx(address, genesisData)
		genesis := Genesis(coinbase)
		fmt.Println("Genesis Block Created")
		err = txn.Set(genesis.Hash, genesis.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), genesis.Hash)
		lastHash = genesis.Hash
		return err

	})
	Handle(err)
	return &BlockChain{lastHash, db}
}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	return &BlockChainIterator{chain.LastHash, chain.Database}
}

func (iter *BlockChainIterator) Next() *Block {
	var block *Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		Handle(err)
		encodedBlock, err := item.Value()
		block = Deserialize(encodedBlock)
		return err
	})
	Handle(err)
	iter.CurrentHash = block.PrevHash
	return block
}

func (chain *BlockChain) FindUTXO() map[string]TxOutputs {
	utxo := make(map[string]TxOutputs)
	spentTxos := make(map[string][]int)

	iter := chain.Iterator()

	for {

		block := iter.Next()

		for _, tx := range block.Transactions {
			txId := hex.EncodeToString(tx.Id)

		Outputs:
			for outId, out := range tx.Outputs {
				if spentTxos[txId] != nil {
					for _, spentOut := range spentTxos[txId] {
						if spentOut == outId {
							continue Outputs
						}
					}
				}
				outs := utxo[txId]
				outs.Outputs = append(outs.Outputs, out)
				utxo[txId] = outs
			}
			if !tx.isCoinbase() {
				for _, in := range tx.Inputs {
					inTxid := hex.EncodeToString(in.Id)
					spentTxos[inTxid] = append(spentTxos[inTxid], in.OutIndex)
				}
			}
		}
		if len(block.PrevHash) == 0 {
			break
		}
	}
	return utxo

}

func (chain *BlockChain) FindTransaction(ID []byte) (Transaction, error) {
	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Equal(tx.Id, ID) {
				return *tx, nil
			}
		}
		if len(block.PrevHash) == 0 {
			break
		}
	}
	return Transaction{}, errors.New("Transaction does not exist")
}

func (chain *BlockChain) SignTransaction(private ecdsa.PrivateKey, tx *Transaction) {
	prevTxs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		inTx, err := chain.FindTransaction(in.Id)
		Handle(err)
		prevTxs[hex.EncodeToString(inTx.Id)] = inTx
	}
	tx.Sign(private, prevTxs)
}

func (chain *BlockChain) VerifyTransaction(tx *Transaction) bool {
	if tx.isCoinbase() {
		return true
	}
	prevTxs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		inTx, err := chain.FindTransaction(in.Id)
		Handle(err)
		prevTxs[hex.EncodeToString(inTx.Id)] = inTx
	}
	return tx.Verify(prevTxs)
}

func DeserializeTransaction(data []byte) Transaction {
	var txn Transaction

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&txn)
	if err != nil {
		Handle(err)
	}
	return txn
}

func retry(dir string, opts badger.Options) (*badger.DB, error) {
	lockPath := filepath.Join(dir, "LOCK")
	if err := os.Remove(lockPath); err != nil {
		return nil, fmt.Errorf("error in removing lock file : %s", err)
	}
	retryOpts := opts
	retryOpts.Truncate = true
	db, err := badger.Open(retryOpts)
	return db, err
}

func openDb(dir string, opts badger.Options) (*badger.DB, error) {
	if db, err := badger.Open(opts); err != nil {
		if strings.Contains(err.Error(), "LOCK") {
			if db, err := retry(dir, opts); err == nil {
				log.Println("Database is unlocked,Value log truncated")
				return db, nil
			} else {
				log.Println("Database could not be unlocked : ", err)
			}
		}
		return nil, err
	} else {
		return db, nil
	}

}
