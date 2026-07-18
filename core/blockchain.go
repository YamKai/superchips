package core

import (
	"fmt"

	"github.com/dgraph-io/badger"
)

const (
	dpPath = "./tmp/blocks"
)

type Blockchain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockchainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func (bc *Blockchain) AddBlock(data string) {
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

	newBlock := NewBlock(data, lastHash)
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

func NewBlockchain() *Blockchain {
	var lastHash []byte
	opts := badger.DefaultOptions(dpPath)
	opts.Dir = dpPath
	opts.ValueDir = dpPath
	db, err := badger.Open(opts)
	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))

		if err == badger.ErrKeyNotFound {
			fmt.Println("No Blockchain found. Generating Genesis...")
			genesis := NewGenesisBlock()

			err = txn.Set(genesis.Hash, genesis.Serialize())
			if err != nil {
				return err
			}

			err = txn.Set([]byte("lh"), genesis.Hash)
			lastHash = genesis.Hash
			return err
		} else if err != nil {
			return err
		} else {
			lastHash, err = item.ValueCopy(nil)
			return err
		}
	})
	Handle(err)

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
