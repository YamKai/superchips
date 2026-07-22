package core

import (
	"crypto/sha256"
	"errors"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/ripemd160"
)

const Version = 0x00

type Wallet struct {
	Mnemonic       string
	MasterKey      *hdkeychain.ExtendedKey
	AddressIndexes map[uint32]string
	NextIndex      uint32
}

func NewHDWallet() (*Wallet, error) {
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return nil, err
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return nil, err
	}
	seed := bip39.NewSeed(mnemonic, "")
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	return &Wallet{mnemonic, masterKey, make(map[uint32]string), 0}, nil
}

func Hash160(b []byte) []byte {
	s := sha256.Sum256(b)
	h := ripemd160.New()
	h.Write(s[:])
	return h.Sum(nil)
}

func CheckSum(b []byte) []byte {
	first := sha256.Sum256(b)
	second := sha256.Sum256(first[:])
	return second[:4]
}

// DeriveAddressAt derives the address string for a specific child index
func (w *Wallet) DeriveAddressAt(index uint32) (string, error) {
	childKey, err := w.DeriveChildKey(index)
	if err != nil {
		return "", err
	}

	pubKey, err := childKey.ECPubKey()
	if err != nil {
		return "", err
	}

	pubBytes := pubKey.SerializeCompressed()
	pubKeyHash := Hash160(pubBytes)
	versionPayload := append([]byte{Version}, pubKeyHash...)
	checkSum := CheckSum(versionPayload)
	address := append(versionPayload, checkSum...)

	return base58.Encode(address), nil
}

// DeriveChildKey navigates m/44'/0'/0'/0/index
func (w *Wallet) DeriveChildKey(index uint32) (*hdkeychain.ExtendedKey, error) {
	purpose, err := w.MasterKey.Derive(hdkeychain.HardenedKeyStart + 44)
	if err != nil {
		return nil, err
	}

	coinType, err := purpose.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return nil, err
	}

	account, err := coinType.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return nil, err
	}

	change, err := account.Derive(0)
	if err != nil {
		return nil, err
	}

	return change.Derive(index)
}

// GenerateNextAddress creates the next address in sequence
func (w *Wallet) GenerateNextAddress(hasHistoryFunc func(string) bool) (string, error) {
	if w.NextIndex > 0 {
		currentAddr := w.AddressIndexes[w.NextIndex-1]
		if !hasHistoryFunc(currentAddr) {
			return "", errors.New("address not found in history function")
		}
	}

	addr, err := w.DeriveAddressAt(w.NextIndex)
	if err != nil {
		return "", err
	}

	w.AddressIndexes[w.NextIndex] = addr
	w.NextIndex++
	return addr, nil
}

// GetPrivateKeyForAddress derives the private key needed to sign a transaction input
func (w *Wallet) GetPrivateKeyForAddress(index uint32) (*btcec.PrivateKey, error) {
	childKey, err := w.DeriveChildKey(index)
	if err != nil {
		return nil, err
	}
	return childKey.ECPrivKey()
}
