package utxodb

import (
	"fmt"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/transaction"
)

const RequestFundsAmount = 1337 // same as Faucet

func (u *UtxoDB) requestFundsTx(target address.Address) (*transaction.Transaction, error) {
	sourceOutputs := u.GetAddressOutputs(u.GetGenesisAddress())

	oids := make([]transaction.OutputID, 0)
	sum := int64(0)
	for oid, bals := range sourceOutputs {
		containsIotas := false
		for _, b := range bals {
			if b.Color == balance.ColorIOTA {
				sum += b.Value
				containsIotas = true
			}
		}
		if containsIotas {
			oids = append(oids, oid)
		}
		if sum >= RequestFundsAmount {
			break
		}
	}
	if sum < RequestFundsAmount {
		return nil, fmt.Errorf("utxodb: not enough input balance")
	}
	inputs := transaction.NewInputs(oids...)

	out := make(map[address.Address][]*balance.Balance)
	out[target] = []*balance.Balance{balance.New(balance.ColorIOTA, RequestFundsAmount)}

	if sum > RequestFundsAmount {
		out[u.GetGenesisAddress()] = []*balance.Balance{balance.New(balance.ColorIOTA, sum-RequestFundsAmount)}
	}

	outputs := transaction.NewOutputs(out)

	tx := transaction.New(inputs, outputs)
	if err := u.CheckInputsOutputs(tx); err != nil {
		return nil, err
	}

	tx.Sign(u.GetGenesisSigScheme())

	if !tx.SignaturesValid() {
		panic("utxodb: something wrong with signatures")
	}

	return tx, nil
}

// RequestFunds implements faucet: it sends 1337 IOTA tokens from genesis to the given address.
func (u *UtxoDB) RequestFunds(target address.Address) (*transaction.Transaction, error) {
	tx, err := u.requestFundsTx(target)
	if err != nil {
		return nil, err
	}

	err = u.AddTransaction(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}
