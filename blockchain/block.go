package blockchain

type Block struct {
	Hash     []byte
	Data     []byte
	PrevHash []byte
	Nonce    int
}

func CreateBlock(data []byte, prevHash []byte) (b *Block) {
	newBlock := &Block{[]byte{}, data, prevHash, 0}
	pow := InitPow(newBlock)

	nonce, hash := pow.Run()
	newBlock.Nonce = nonce
	newBlock.Hash = hash[:]

	return newBlock
}

func Genesis() (b *Block) {
	newBlock := CreateBlock([]byte("Genesis"), []byte{})
	return newBlock
}

type BlockChain struct {
	Blocks []*Block
}

func (chain *BlockChain) AddBlock(data string) {
	lastBlock := chain.Blocks[len(chain.Blocks)-1]
	newBlock := CreateBlock([]byte(data), lastBlock.Hash)
	chain.Blocks = append(chain.Blocks, newBlock)
}

func InitBlockChain() *BlockChain {
	newChain := &BlockChain{[]*Block{Genesis()}}
	return newChain
}
