package wallet

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"log"
	"os"
)

const walletFile = "./tmp/wallets.data"

type Wallets struct {
	Wallets map[string]*Wallet
}

func CreateWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)
	err := wallets.loadFile()
	return &wallets, err
}
func (ws *Wallets) AddWallet() string {
	wallet := MakeWallet()
	address := string(wallet.Address())
	ws.Wallets[address] = wallet
	return address
}

func (ws *Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}
func (ws *Wallets) GetAllAddresses() []string {
	var addresses []string
	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}
	return addresses
}
func (ws *Wallets) loadFile() error {

	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}
	var wallets Wallets
	content, err := os.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(content))
	err = decoder.Decode(&wallets)

	if err != nil {
		log.Panic(err)
	}

	ws.Wallets = wallets.Wallets
	return nil

}

func (ws *Wallets) SaveFile() {
	var buf bytes.Buffer

	gob.Register(elliptic.P256())
	encoder := gob.NewEncoder(&buf)

	err := encoder.Encode(ws)

	if err != nil {
		log.Panic(err)
	}
	err = os.WriteFile(walletFile, buf.Bytes(), 0644)

	if err != nil {
		log.Panic(err)
	}

}
