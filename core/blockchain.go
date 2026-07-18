package core

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/dgraph-io/badger"
)

const (
	dbPath      = "./tmp/blocks"
	dbFile      = "./tmp/blocks/MANIFEST"
	genesisData = "First Transaction from Genesis"
)

type Blockchain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockchainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func (bc *Blockchain) AddBlock(transactions []*Transaction) {
	var lastHash []byte

	err := bc.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		if err != nil {
			return err
		}
		lastHash, err = item.ValueCopy(nil)
		return err
	})
	Handle(err)

	newBlock := NewBlock(transactions, lastHash)
	err = bc.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			return err
		}
		err = txn.Set([]byte("lh"), newBlock.Hash)
		bc.LastHash = newBlock.Hash
		return err
	})
	Handle(err)
}

func NewBlockchain(address string) *Blockchain {
	var lastHash []byte

	if DBexists() {
		log.Fatalf("Blockchain creation aborted: A database already exists at %s\n", dbPath)
	}

	opts := badger.DefaultOptions(dbPath)
	opts.Dir = dbPath
	opts.ValueDir = dbPath
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Explicitly handle the transaction return cleanly
	err = db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(address, genesisData)
		genesis := NewGenesisBlock(cbtx)
		fmt.Println("Genesis object created in memory.")

		// Use local, isolated variables instead of re-assigning the outer 'err'
		if setErr := txn.Set(genesis.Hash, genesis.Serialize()); setErr != nil {
			return setErr
		}

		if setErr := txn.Set([]byte("lh"), genesis.Hash); setErr != nil {
			return setErr
		}

		lastHash = genesis.Hash
		return nil // Transaction successfully completed!
	})

	if err != nil {
		db.Close() // Safe closure to clear locks
		log.Fatalf("Transaction failed: %v", err)
	}

	blockchain := Blockchain{lastHash, db}
	return &blockchain
}

func (bc *Blockchain) Iterator() *BlockchainIterator {
	return &BlockchainIterator{bc.LastHash, bc.Database}
}

func (iter *BlockchainIterator) Next() *Block {
	var block *Block
	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		if err != nil {
			return err
		}
		encodedBlock, err := item.ValueCopy(nil)
		block = DeserializeBlock(encodedBlock)
		return err
	})
	Handle(err)

	iter.CurrentHash = block.PrevBlockHash
	return block
}

func ContinueBlockChain(address string) *Blockchain {
	if !DBexists() {
		log.Fatalf("Error: Blockchain database does not exist at %s. Run 'createblockchain' first.\n", dbPath)
	}

	var lastHash []byte
	opts := badger.DefaultOptions(dbPath)
	opts.Dir = dbPath
	opts.ValueDir = dbPath
	db, err := badger.Open(opts)
	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		if err != nil {
			return err
		}
		lastHash, err = item.ValueCopy(nil)
		return err
	})
	Handle(err)

	blockchain := Blockchain{lastHash, db}
	return &blockchain
}

func DBexists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

func (blockchain *Blockchain) FindUnspentTransaction(address string) []Transaction {
	var unspentTxs []Transaction
	spentTxOutputs := make(map[string][]int)
	iter := blockchain.Iterator()
	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)
		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTxOutputs[txID] != nil {
					for _, spentOut := range spentTxOutputs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				if out.CanBeUnlocked(address) {
					unspentTxs = append(unspentTxs, *tx)
				}
			}
			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					if in.CanUnlock(address) {
						inTxID := hex.EncodeToString(in.ID)
						spentTxOutputs[inTxID] = append(spentTxOutputs[inTxID], in.Out)
					}
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return unspentTxs
}

func (blockchain *Blockchain) FindUTXO(address string) []TxOutput {
	var UTXOs []TxOutput
	unspentTX := blockchain.FindUnspentTransaction(address)
	for _, tx := range unspentTX {
		for _, out := range tx.Outputs {
			if out.CanBeUnlocked(address) {
				UTXOs = append(UTXOs, out)
			}
		}
	}
	return UTXOs
}

func (blockchain *Blockchain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTX := blockchain.FindUnspentTransaction(address)
	accumulated := 0

Work:
	for _, tx := range unspentTX {
		txID := hex.EncodeToString(tx.ID)
		for outIdx, out := range tx.Outputs {
			if out.CanBeUnlocked(address) && accumulated < amount {
				accumulated += out.Value
				unspentOuts[txID] = append(unspentOuts[txID], outIdx)
				if accumulated >= amount {
					break Work
				}
			}
		}
	}
	return accumulated, unspentOuts
}
