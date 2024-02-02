package blockchain

import (
	"bytes"
	"encoding/hex"
	"log"

	"github.com/dgraph-io/badger"
)

type UTXOSet struct {
	Blockchain *BlockChain
}

var (
	utxoPrefix = []byte("utxo-")
)

func (u UTXOSet) FindUTXO(pubKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput

	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			v, err := item.Value()
			Handle(err)
			outs := DeserializeTxOutputs(v)

			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}

		return nil
	})
	Handle(err)

	return UTXOs
}
func (utxo UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	accumulated := 0

	db := utxo.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		for iter.Seek(utxoPrefix); iter.ValidForPrefix(utxoPrefix); iter.Next() {
			item := iter.Item()
			key := item.Key()
			key = bytes.TrimPrefix(key, utxoPrefix)
			v, err := item.Value()
			Handle(err)
			outputs := DeserializeTxOutputs(v)
			txId := hex.EncodeToString(key)
			for outId, out := range outputs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
					accumulated += out.Value
					unspentOuts[txId] = append(unspentOuts[txId], outId)
				}
			}

		}
		return nil
	})
	Handle(err)
	return accumulated, unspentOuts

}
func (utxo UTXOSet) ReIndex() {
	db := utxo.Blockchain.Database
	utxo.DeleteByPrefix(utxoPrefix)
	UTXO := utxo.Blockchain.FindUTXO()
	err := db.Update(func(txn *badger.Txn) error {
		for txId, outs := range UTXO {
			key, err := hex.DecodeString(txId)
			if err != nil {
				return err
			}
			key = append(utxoPrefix, key...)
			err = txn.Set(key, outs.Serialize())
			Handle(err)
		}
		return nil
	})
	Handle(err)
}

func (utxo UTXOSet) Update(block *Block) {
	db := utxo.Blockchain.Database

	err := db.Update(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions {
			if !tx.isCoinbase() {
				for _, in := range tx.Inputs {
					UpdatedOutputs := TxOutputs{}
					inId := append(utxoPrefix, in.Id...)
					item, err := txn.Get(inId)
					Handle(err)
					v, err := item.Value()
					Handle(err)
					outputs := DeserializeTxOutputs(v)

					for outId, out := range outputs.Outputs {
						if outId != in.OutIndex {
							UpdatedOutputs.Outputs = append(UpdatedOutputs.Outputs, out)
						}
					}

					if len(UpdatedOutputs.Outputs) == 0 {
						err := txn.Delete(inId)
						Handle(err)
					} else {
						if err := txn.Set(inId, UpdatedOutputs.Serialize()); err != nil {
							log.Panic(err)
						}
					}

				}
			}
			newOutputs := TxOutputs{}

			newOutputs.Outputs = append(newOutputs.Outputs, tx.Outputs...)
			prefix := append(utxoPrefix, tx.Id...)
			if err := txn.Set(prefix, newOutputs.Serialize()); err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
	Handle(err)
}
func (utxo UTXOSet) CountTransactions() int {
	db := utxo.Blockchain.Database
	count := 0
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		iter := txn.NewIterator(opts)
		defer iter.Close()
		for iter.Seek(utxoPrefix); iter.ValidForPrefix(utxoPrefix); iter.Next() {
			count++
		}
		return nil
	})
	Handle(err)
	return count
}
func (utxo *UTXOSet) DeleteByPrefix(prefix []byte) {
	deleteKeys := func(keyList [][]byte) error {
		if err := utxo.Blockchain.Database.Update(func(txn *badger.Txn) error {
			for _, key := range keyList {
				if err := txn.Delete(key); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}

	collectSize := 100000
	utxo.Blockchain.Database.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		iter := txn.NewIterator(opts)
		defer iter.Close()
		keysList := make([][]byte, 0, collectSize)
		keysCollected := 0
		for iter.Seek(prefix); iter.ValidForPrefix(prefix); iter.Next() {
			key := iter.Item().KeyCopy(nil)
			keysList = append(keysList, key)
			keysCollected++
			if keysCollected == collectSize {
				if err := deleteKeys(keysList); err != nil {
					log.Panic(err)
				}
				keysList = make([][]byte, 0, collectSize)
				keysCollected = 0
			}
		}
		if keysCollected > 0 {
			if err := deleteKeys(keysList); err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
}
