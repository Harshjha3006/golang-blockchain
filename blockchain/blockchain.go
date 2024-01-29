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

func (chain *BlockChain) AddBlock(txs []*Transaction) {
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

func (chain *BlockChain) FindUnspentTransactions(pubKeyHash []byte) []*Transaction {
	var unspentTxns []*Transaction
	spentTxOs := make(map[string][]int)

	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, txn := range block.Transactions {
			txId := hex.EncodeToString(txn.Id)

		Output:
			for outIndex, out := range txn.Outputs {
				if spentTxOs[txId] != nil {
					for _, spentOut := range spentTxOs[txId] {
						if spentOut == outIndex {
							continue Output
						}
					}
				}
				if out.IsLockedWithKey(pubKeyHash) {
					unspentTxns = append(unspentTxns, txn)
				}
			}

			if !txn.isCoinbase() {
				for _, in := range txn.Inputs {
					if in.CanUseKey(pubKeyHash) {
						inTxnId := hex.EncodeToString(in.Id)
						spentTxOs[inTxnId] = append(spentTxOs[inTxnId], in.OutIndex)
					}
				}
			}
		}
		if len(block.PrevHash) == 0 {
			break
		}
	}
	return unspentTxns

}

func (chain *BlockChain) FindUTXO(pubKeyHash []byte) []TxOutput {
	var utxo []TxOutput
	unspentTransactions := chain.FindUnspentTransactions(pubKeyHash)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.IsLockedWithKey(pubKeyHash) {
				utxo = append(utxo, out)
			}
		}
	}
	return utxo
}

func (chain *BlockChain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxns := chain.FindUnspentTransactions(pubKeyHash)
	accumulated := 0

Work:
	for _, txn := range unspentTxns {
		txId := hex.EncodeToString(txn.Id)
		for outIdx, out := range txn.Outputs {
			if out.IsLockedWithKey(pubKeyHash) {
				accumulated += out.Value
				unspentOuts[txId] = append(unspentOuts[txId], outIdx)
				if accumulated >= amount {
					break Work
				}
			}
		}
	}
	return accumulated, unspentOuts
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
