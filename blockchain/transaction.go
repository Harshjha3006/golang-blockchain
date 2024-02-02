package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/Harshjha3006/golang-blockchain/wallet"
)

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
	txinput := TxInput{[]byte{}, -1, nil, []byte(data)}
	txoutput := *NewTXOutput(to, 100)

	tx := Transaction{nil, []TxInput{txinput}, []TxOutput{txoutput}}

	tx.setId()

	return &tx

}

func (tx *Transaction) isCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].Id) == 0 && tx.Inputs[0].OutIndex == -1
}

func NewTransaction(from string, to string, amount int, utxo UTXOSet) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	wallets, err := wallet.CreateWallets()
	Handle(err)
	w := wallets.GetWallet(from)
	pubKeyHash := wallet.PubkeyHash(w.PublicKey)
	acc, validOpts := utxo.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		log.Panic("Err : Not enough funds")
	}

	for txId, out := range validOpts {
		txId, err := hex.DecodeString(txId)
		Handle(err)

		for outIdx := range out {
			input := TxInput{txId, outIdx, nil, w.PublicKey}
			inputs = append(inputs, input)
		}
	}
	outputs = append(outputs, *NewTXOutput(to, amount))

	if acc > amount {
		outputs = append(outputs, *NewTXOutput(from, acc-amount))
	}
	tx := Transaction{nil, inputs, outputs}
	tx.setId()
	utxo.Blockchain.SignTransaction(w.PrivateKey, &tx)

	return &tx
}

func (tx *Transaction) Sign(private ecdsa.PrivateKey, prevTxs map[string]Transaction) {
	if tx.isCoinbase() {
		return
	}
	for _, in := range tx.Inputs {
		if prevTxs[hex.EncodeToString(in.Id)].Id == nil {
			log.Panic("Error : Previous Transaction does not exists")
		}
	}
	txCopy := tx.TrimmedCopy()

	for inId, in := range txCopy.Inputs {
		prevTx := prevTxs[hex.EncodeToString(in.Id)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTx.Outputs[in.OutIndex].PubKeyHash
		txCopy.setId()
		txCopy.Inputs[inId].PubKey = nil
		r, s, err := ecdsa.Sign(rand.Reader, &private, txCopy.Id)
		Handle(err)
		signature := append(r.Bytes(), s.Bytes()...)
		tx.Inputs[inId].Signature = signature
	}

}

func (tx *Transaction) Verify(private ecdsa.PrivateKey, prevTxs map[string]Transaction) bool {
	if tx.isCoinbase() {
		return true
	}
	for _, in := range tx.Inputs {
		if prevTxs[hex.EncodeToString(in.Id)].Id == nil {
			log.Panic("Error : Previous Transaction does not exists")
		}
	}
	txCopy := tx.TrimmedCopy()

	for inId, in := range txCopy.Inputs {
		prevTx := prevTxs[hex.EncodeToString(in.Id)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTx.Outputs[in.OutIndex].PubKeyHash
		txCopy.setId()
		txCopy.Inputs[inId].PubKey = nil

		curve := elliptic.P256()
		r := big.Int{}
		s := big.Int{}

		signlen := len(in.Signature)
		r.SetBytes(in.Signature[:(signlen / 2)])
		s.SetBytes(in.Signature[(signlen / 2):])

		x := big.Int{}
		y := big.Int{}

		pubLen := len(in.PubKey)
		x.SetBytes(in.PubKey[:(pubLen / 2)])
		y.SetBytes(in.PubKey[(pubLen / 2):])

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

		if !ecdsa.Verify(&rawPubKey, txCopy.Id, &r, &s) {
			return false
		}
	}
	return true
}
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	for _, in := range tx.Inputs {
		inputs = append(inputs, TxInput{in.Id, in.OutIndex, nil, nil})
	}

	for _, out := range tx.Outputs {
		outputs = append(outputs, TxOutput{out.Value, out.PubKeyHash})

	}

	return Transaction{tx.Id, inputs, outputs}
}

func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.Id))
	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:     %x", input.Id))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.OutIndex))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}
