package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"superchips/core"
)

type CommandLine struct {
	blockchain *core.Blockchain
}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("add -block block data")
	fmt.Println("print - print into the blockchain")
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) addBlock(data string) {
	cli.blockchain.AddBlock(data)
	fmt.Println("added block.")
}

func (cli *CommandLine) printChain() {
	iter := cli.blockchain.Iterator()
	for {
		block := iter.Next()
		fmt.Printf("Prev. Hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Data:       %s\n", block.Data)
		fmt.Printf("Block Hash: %x\n", block.Hash)
		fmt.Printf("Nonce:      %d\n", block.Nonce)
		pow := core.NewProof(block)
		fmt.Printf("PoW:%s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) run() {
	cli.validateArgs()
	addBlockCmd := flag.NewFlagSet("add", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)
	addBlockData := addBlockCmd.String("block", "", "block data")
	switch os.Args[1] {
	case "add":
		err := addBlockCmd.Parse(os.Args[2:])
		core.Handle(err)
	case "print":
		err := printChainCmd.Parse(os.Args[2:])
		core.Handle(err)
	default:
		cli.printUsage()
		runtime.Goexit()
	}
	if addBlockCmd.Parsed() {
		if *addBlockData == "" {
			addBlockCmd.Usage()
			runtime.Goexit()
		} else {
			cli.addBlock(*addBlockData)
		}
	}
	if printChainCmd.Parsed() {
		cli.printChain()
	}
}

func main() {
	defer os.Exit(0)
	bc := core.NewBlockchain()
	defer bc.Database.Close()
	cli := CommandLine{bc}
	cli.run()
}
