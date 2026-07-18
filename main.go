package main

import (
	"fmt"
	"strconv"
	"superchips/core"
)

func main() {
	bc := core.NewBlockchain()
	bc.AddBlock("Send 3.1 Super to Alice")
	bc.AddBlock("Send 100 Chips to Bob")

	for i, block := range bc.Blocks {
		fmt.Printf("--- Block %d ---\n", i)
		fmt.Printf("Prev. Hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Data:       %s\n", block.Data)
		fmt.Printf("Block Hash: %x\n", block.Hash)
		fmt.Printf("Nonce:      %d\n", block.Nonce)
		pow := core.NewProof(block)
		fmt.Printf("PoW:%s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()
	}
}
