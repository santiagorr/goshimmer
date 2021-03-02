package utxodb

import (
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/address"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBasic(t *testing.T) {
	u := New()
	genTx, ok := u.GetTransaction(u.genesisTxId)
	assert.Equal(t, ok, true)
	assert.Equal(t, genTx.ID(), u.genesisTxId)
}

func getBalance(u *UtxoDB, address address.Address) int64 {
	gout := u.GetAddressOutputs(address)
	total := int64(0)
	for oid := range gout {
		sum, err := u.getOutputTotal(oid)
		if err != nil {
			panic(err)
		}
		total += sum
	}
	return total
}

func TestGenesis(t *testing.T) {
	u := New()
	assert.Equal(t, supply, getBalance(u, u.GetGenesisSigScheme().Address()))
	u.checkLedgerBalance()
}

func TestRequestFunds(t *testing.T) {
	u := New()
	addr := NewSigScheme("C6hPhCS2E2dKUGS3qj4264itKXohwgL3Lm2fNxayAKr", 0).Address()
	_, err := u.RequestFunds(addr)
	assert.NoError(t, err)
	assert.EqualValues(t, supply-RequestFundsAmount, getBalance(u, u.GetGenesisSigScheme().Address()))
	assert.EqualValues(t, RequestFundsAmount, getBalance(u, addr))
	u.checkLedgerBalance()
}

func TestTransferAndBook(t *testing.T) {
	u := New()

	addr := NewSigScheme("C6hPhCS2E2dKUGS3qj4264itKXohwgL3Lm2fNxayAKr", 0).Address()
	tx, err := u.RequestFunds(addr)
	assert.NoError(t, err)
	assert.EqualValues(t, supply-RequestFundsAmount, getBalance(u, u.GetGenesisSigScheme().Address()))
	assert.EqualValues(t, RequestFundsAmount, getBalance(u, addr))
	u.checkLedgerBalance()

	err = u.AddTransaction(tx)
	assert.Error(t, err)
}
