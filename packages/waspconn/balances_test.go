// +build ignorepackage waspconn

import (
	"bytes"
	"testing"

	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/transaction"
	"github.com/stretchr/testify/assert"
)

func TestBalances(t *testing.T) {
	bals := map[transaction.ID][]*balance.Balance{
		transaction.RandomID(): {
			balance.New(balance.ColorIOTA, 42),
		},
	}

	var buf bytes.Buffer
	err := WriteBalances(&buf, bals)
	assert.Equal(t, err, nil)

	balsBack, err := ReadBalances(bytes.NewReader(buf.Bytes()))
	assert.Equal(t, err, nil)

	var bufBack bytes.Buffer
	err = WriteBalances(&bufBack, balsBack)

	assert.Equal(t, bytes.Equal(buf.Bytes(), bufBack.Bytes()), true)

	assert.Equal(t, err, nil)

	_ = BalancesToOutputs(address.Address{}, balsBack)
}
