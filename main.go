package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"superchips/core"
)

type CommandLine struct{}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  balance -address ADDRESS          - get the balance of an address")
	fmt.Println("  createblockchain -address ADDRESS - creates a new blockchain")
	fmt.Println("  print                             - print into the blockchain")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT - Send amount")
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CommandLine) printChain() {
	blockchain := core.ContinueBlockChain("")
	defer blockchain.Database.Close()
	iter := blockchain.Iterator()
	for {
		block := iter.Next()
		fmt.Printf("Prev. Hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Block Hash: %x\n", block.Hash)
		pow := core.NewProof(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println("----------------------------------------")
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) createBlockchain(address string) {
	blockchain := core.NewBlockchain(address)
	blockchain.Database.Close()
	fmt.Println("Created blockchain successfully.")
}

func (cli *CommandLine) getBalance(address string) {
	blockchain := core.ContinueBlockChain("")
	defer blockchain.Database.Close()
	balance := 0
	UTXOs := blockchain.FindUTXO(address)
	for _, out := range UTXOs {
		balance += out.Value
	}
	fmt.Printf("Balance of '%s': %d\n", address, balance)
}

func (cli *CommandLine) send(from, to string, amount int) {
	blockchain := core.ContinueBlockChain(from)
	defer blockchain.Database.Close()
	tx := core.NewTransaction(from, to, amount, blockchain)
	blockchain.AddBlock([]*core.Transaction{tx})
	fmt.Println("Added ", amount, "from", from, " to ", to)
}

func (cli *CommandLine) run() {
	cli.validateArgs()

	getBalanceCmd := flag.NewFlagSet("balance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "the address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "the address to createblockchain for")
	sendFrom := sendCmd.String("from", "", "the address to send from")
	sendTo := sendCmd.String("to", "", "the address to send to")
	sendAmount := sendCmd.Int("amount", 0, "the amount to send")

	switch os.Args[1] {
	case "balance":
		err := getBalanceCmd.Parse(os.Args[2:])
		core.Handle(err)
	case "print":
		err := printChainCmd.Parse(os.Args[2:])
		core.Handle(err)
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		core.Handle(err)
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		core.Handle(err)
	default:
		cli.printUsage()
		os.Exit(1)
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddress)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			os.Exit(1)
		}
		cli.createBlockchain(*createBlockchainAddress)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount == 0 {
			sendCmd.Usage()
			os.Exit(1)
		}
		cli.send(*sendFrom, *sendTo, *sendAmount)
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}
}

func main() {
	cli := CommandLine{}
	cli.run()
}
