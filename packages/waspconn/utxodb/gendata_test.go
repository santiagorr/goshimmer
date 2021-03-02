package utxodb

import (
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/address/signaturescheme"
	"github.com/iotaledger/hive.go/crypto/ed25519"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGenSeed(t *testing.T) {
	seed := ed25519.NewSeed()
	seedStr := base58.Encode(seed.Bytes())
	t.Logf("seed: %s", seedStr)

	seedBin, err := base58.Decode(seedStr)
	assert.NoError(t, err)

	seedRecover := ed25519.NewSeed(seedBin)
	seedRecoverStr := base58.Encode(seedRecover.Bytes())
	assert.EqualValues(t, seedStr, seedRecoverStr)
}

func TestGenOwnerAddress(t *testing.T) {
	for i := 4; i <= 10; i++ {
		keyPair := ed25519.GenerateKeyPair()
		t.Logf("ownerPrivKey%d = \"%s\"", i, keyPair.PrivateKey.String())
		t.Logf("ownerPubKey%d = \"%s\"", i, keyPair.PublicKey.String())
		sigscheme := signaturescheme.ED25519(keyPair)
		t.Logf("ownerAddress%d = \"%s\"", i, sigscheme.Address().String())
	}
}
