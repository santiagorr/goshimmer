package utxodb

import (
	"errors"
	"fmt"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/transaction"
)

func (u *UtxoDB) collectInputBalances(tx *transaction.Transaction) (map[balance.Color]int64, int64, error) {
	ret := make(map[balance.Color]int64)
	retsum := int64(0)

	var err error
	tx.Inputs().ForEach(func(outputId transaction.OutputID) bool {
		txInp, ok := u.GetTransaction(outputId.TransactionID())
		if !ok {
			err = fmt.Errorf("can't find txid %s", outputId.TransactionID().String())
			return false
		}
		balances, ok := txInp.Outputs().Get(outputId.Address())
		if !ok {
			err = fmt.Errorf("can't find address %s among outputs of txid %s",
				outputId.Address().String(), outputId.TransactionID().String())
			return false
		}
		for _, bal := range balances.([]*balance.Balance) {
			col := bal.Color
			if col == balance.ColorNew {
				col = (balance.Color)(txInp.ID())
			}
			if _, ok := ret[col]; !ok {
				ret[col] = 0
			}
			ret[col] = ret[col] + bal.Value
			retsum += bal.Value
		}
		return true
	})
	if err != nil {
		return nil, 0, err
	}
	return ret, retsum, nil
}

func collectOutputBalances(tx *transaction.Transaction) (map[balance.Color]int64, int64) {
	ret := make(map[balance.Color]int64)
	retsum := int64(0)

	tx.Outputs().ForEach(func(_ address.Address, balances []*balance.Balance) bool {
		for _, bal := range balances {
			if _, ok := ret[bal.Color]; !ok {
				ret[bal.Color] = 0
			}
			ret[bal.Color] = ret[bal.Color] + bal.Value
			retsum += bal.Value
		}
		return true
	})
	return ret, retsum
}

func (u *UtxoDB) CheckInputsOutputs(tx *transaction.Transaction) error {
	inbals, insum, err := u.collectInputBalances(tx)
	if err != nil {
		return fmt.Errorf("utxodb.CheckInputsOutputs: wrong inputs: %v", err)
	}
	outbals, outsum := collectOutputBalances(tx)
	if insum != outsum {
		return errors.New("utxodb.CheckInputsOutputs unequal totals")
	}

	for col, inb := range inbals {
		if !(col != balance.ColorNew) {
			return errors.New("utxodb.CheckInputsOutputs: assertion failed: col != balance.ColorNew")
		}
		if col == balance.ColorIOTA {
			continue
		}
		outb, ok := outbals[col]
		if !ok {
			continue
		}
		if outb > inb {
			// colored supply can't be inflated
			return errors.New("utxodb.CheckInputsOutputs: colored supply can't be inflated")
		}
	}
	return nil
}
