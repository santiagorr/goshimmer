package utxodb

import (
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/address/signaturescheme"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/transaction"
	"github.com/iotaledger/hive.go/crypto/ed25519"
	"github.com/mr-tron/base58"
)

const (
	supply = int64(100 * 1000 * 1000 * 1000)
)

var (
	genesisSigScheme = NewSigScheme("EFonzaUz5ngYeDxbRKu8qV5aoSogUQ5qVSTSjn7hJ8FQ", 0)
)

// NewSigScheme creates new random Ed25519 signature scheme
func NewSigScheme(seedStr string, index int) signaturescheme.SignatureScheme {
	seedBin, err := base58.Decode(seedStr)
	if err != nil {
		panic(err)
	}
	seed := ed25519.NewSeed(seedBin)
	keyPair := seed.KeyPair(uint64(index))
	return signaturescheme.ED25519(*keyPair)
}

func (u *UtxoDB) genesisInit() {
	// create genesis transaction
	genesisAddr := u.GetGenesisAddress()
	genesisInput := transaction.NewOutputID(genesisAddr, transaction.ID{})
	inputs := transaction.NewInputs(genesisInput)
	outputs := transaction.NewOutputs(map[address.Address][]*balance.Balance{
		genesisAddr: {balance.New(balance.ColorIOTA, supply)},
	})
	genesisTx := transaction.New(inputs, outputs)
	genesisTx.Sign(u.GetGenesisSigScheme())

	u.genesisTxId = genesisTx.ID()

	u.transactions[u.genesisTxId] = genesisTx
	u.utxo[transaction.NewOutputID(genesisAddr, u.genesisTxId)] = true
	u.utxoByAddress[genesisAddr] = []transaction.ID{u.genesisTxId}
}

// GetGenesisSigScheme return signature scheme used by creator of genesis
func (u *UtxoDB) GetGenesisSigScheme() signaturescheme.SignatureScheme {
	return genesisSigScheme
}

// GetGenesisAddress return address of genesis
func (u *UtxoDB) GetGenesisAddress() address.Address {
	return genesisSigScheme.Address()
}
