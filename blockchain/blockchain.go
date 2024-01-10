package blockchain

import (
	"fmt"

	"github.com/dgraph-io/badger/v4"
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
	dbPath = "tmp/blocks"
)

func (chain *BlockChain) AddBlock(data string) {
	var lastHash []byte
	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		err = item.Value(func(val []byte) error {
			lastHash = append(lastHash, val...)
			return nil
		})
		return err
	})
	Handle(err)
	newBlock := CreateBlock([]byte(data), lastHash)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err = txn.Set([]byte("lh"), newBlock.Hash)
		Handle(err)
		err = txn.Set(newBlock.Hash, newBlock.Serialize())
		return err
	})
	Handle(err)

}

func InitBlockChain() *BlockChain {
	var lastHash []byte
	db, err := badger.Open(badger.DefaultOptions(dbPath))
	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get([]byte("lh")); err == badger.ErrKeyNotFound {
			fmt.Println("No existing blockchain found")
			genesis := Genesis()
			fmt.Println("Genesis Proved")
			err = txn.Set(genesis.Hash, genesis.Serialize())
			Handle(err)
			err = txn.Set([]byte("lh"), genesis.Hash)
			return err
		} else {
			item, err := txn.Get([]byte("lh"))
			Handle(err)
			err = item.Value(func(val []byte) error {
				lastHash = append(lastHash, val...)
				return nil
			})
			return err
		}
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
		err = item.Value(func(val []byte) error {
			block = Deserialize(append([]byte{}, val...))
			return nil
		})
		return err
	})
	Handle(err)
	iter.CurrentHash = block.PrevHash
	return block
}
