package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
)

type TxOutput struct {
	Value  int
	PubKey string
}

type TxInput struct {
	Id       []byte
	OutIndex int
	Sig      string
}

type Transaction struct {
	Id      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

func (tx *Transaction) setId() {
	var buffer bytes.Buffer
	var hash [32]byte
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(tx)
	Handle(err)
	hash = sha256.Sum256(buffer.Bytes())
	tx.Id = hash[:]

}

func CoinbaseTx(to string, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}
	txinput := TxInput{[]byte{}, -1, data}
	txoutput := TxOutput{100, to}

	tx := Transaction{nil, []TxInput{txinput}, []TxOutput{txoutput}}

	tx.setId()

	return &tx

}

func (tx *Transaction) isCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].Id) == 0 && tx.Inputs[0].OutIndex == -1
}

func (in *TxInput) canUnlock(data string) bool {
	return in.Sig == data
}

func (out *TxOutput) canBeUnlocked(data string) bool {
	return out.PubKey == data
}

func NewTransaction(from string, to string, amount int, chain *BlockChain) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	acc, validOpts := chain.FindSpendableOutputs(from, amount)

	if acc < amount {
		log.Panic("Err : Not enough funds")
	}

	for txId, out := range validOpts {
		txId, err := hex.DecodeString(txId)
		Handle(err)

		for outIdx := range out {
			input := TxInput{txId, outIdx, from}
			inputs = append(inputs, input)
		}
	}
	output := TxOutput{amount, to}
	outputs = append(outputs, output)

	if acc > amount {
		outputs = append(outputs, TxOutput{acc - amount, from})
	}
	tx := Transaction{nil, inputs, outputs}
	tx.setId()
	return &tx
}
