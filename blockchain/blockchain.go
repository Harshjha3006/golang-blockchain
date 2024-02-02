package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"runtime"

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
	dbPath      = "./tmp/blocks"
	dbFile      = "./tmp/blocks/MANIFEST"
	genesisData = "First Transaction from Genesis"
)

func (chain *BlockChain) AddBlock(txs []*Transaction) *Block {
	var lastHash []byte
	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		lastHash, err = item.Value()
		return err
	})
	Handle(err)
	newBlock := CreateBlock(txs, lastHash)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err = txn.Set([]byte("lh"), newBlock.Hash)
		Handle(err)
		err = txn.Set(newBlock.Hash, newBlock.Serialize())
		return err
	})
	Handle(err)
	return newBlock
}

func DbExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

func ContinueBlockChain() *BlockChain {

	if !DbExists() {
		fmt.Println("No BlockChain found, Create one")
		runtime.Goexit()
	}

	var lastHash []byte
	opts := badger.DefaultOptions
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	db, err := badger.Open(opts)
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

func InitBlockChain(address string) *BlockChain {

	if DbExists() {
		fmt.Println("BlockChain already exists")
		runtime.Goexit()
	}

	var lastHash []byte
	opts := badger.DefaultOptions
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	db, err := badger.Open(opts)
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

func (chain *BlockChain) VerifyTransaction(private ecdsa.PrivateKey, tx *Transaction) {
	prevTxs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		inTx, err := chain.FindTransaction(in.Id)
		Handle(err)
		prevTxs[hex.EncodeToString(inTx.Id)] = inTx
	}
	tx.Verify(private, prevTxs)
}
