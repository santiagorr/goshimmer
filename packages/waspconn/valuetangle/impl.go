package valuetangle

import (
	"fmt"

	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"github.com/iotaledger/goshimmer/packages/tangle"
	"github.com/iotaledger/goshimmer/plugins/config"
	"github.com/iotaledger/goshimmer/plugins/faucet"
	"github.com/iotaledger/goshimmer/plugins/issuer"
	"github.com/iotaledger/goshimmer/plugins/messagelayer"
	"github.com/iotaledger/hive.go/events"
)

// ValueTangle imlpements waspconn.ValueTangle with the Goshimmer tangle as backend
type ValueTangle struct {
	txConfirmedClosure  *events.Closure
	txConfirmedCallback func(tx *ledgerstate.Transaction)

	txBookedClosure  *events.Closure
	txBookedCallback func(tx *ledgerstate.Transaction)
}

func extractTransaction(id tangle.MessageID, f func(*ledgerstate.Transaction)) {
	if f == nil {
		return
	}
	messagelayer.Tangle().Storage.Message(id).Consume(func(msg *tangle.Message) {
		if payload := msg.Payload(); payload != nil && payload.Type() == ledgerstate.TransactionType {
			f(payload.(*ledgerstate.Transaction))
		}
	})
}

// New returns an implementation for waspconn.ValueTangle
func New() *ValueTangle {
	v := &ValueTangle{}

	v.txConfirmedClosure = events.NewClosure(func(id tangle.MessageID) {
		extractTransaction(id, v.txConfirmedCallback)
	})
	messagelayer.Tangle().OpinionFormer.Events.TransactionConfirmed.Attach(v.txConfirmedClosure)

	v.txBookedClosure = events.NewClosure(func(id tangle.MessageID) {
		extractTransaction(id, v.txBookedCallback)
	})
	messagelayer.Tangle().Booker.Events.MessageBooked.Attach(v.txBookedClosure)

	return v
}

func (v *ValueTangle) Detach() {
	messagelayer.Tangle().OpinionFormer.Events.TransactionConfirmed.Detach(v.txConfirmedClosure)
	messagelayer.Tangle().Booker.Events.MessageBooked.Detach(v.txBookedClosure)
}

func (v *ValueTangle) OnTransactionConfirmed(cb func(tx *ledgerstate.Transaction)) {
	v.txConfirmedCallback = cb
}

func (v *ValueTangle) OnTransactionBooked(cb func(tx *ledgerstate.Transaction)) {
	v.txBookedCallback = cb
}

// GetAddressOutputs returns the available UTXOs for an address
func (v *ValueTangle) GetAddressOutputs(addr ledgerstate.Address) map[ledgerstate.OutputID]*ledgerstate.ColoredBalances {
	ret := make(map[ledgerstate.OutputID]*ledgerstate.ColoredBalances)
	messagelayer.Tangle().LedgerState.OutputsOnAddress(addr).Consume(func(output ledgerstate.Output) {
		messagelayer.Tangle().LedgerState.OutputMetadata(output.ID()).Consume(func(outputMetadata *ledgerstate.OutputMetadata) {
			if outputMetadata.Finalized() && outputMetadata.ConsumerCount() == 0 {
				ret[output.ID()] = output.Balances()
			}
		})
	})
	return ret
}

// GetTransaction fetches a transaction by ID, and executes the given callback if found
func (v *ValueTangle) GetTransaction(txid ledgerstate.TransactionID, f func(ret *ledgerstate.Transaction)) (found bool) {
	found = false
	messagelayer.Tangle().LedgerState.Transaction(txid).Consume(func(tx *ledgerstate.Transaction) {
		found = true
		f(tx)
	})
	return
}

func (ce *ValueTangle) GetBranchInclusionState(txid ledgerstate.TransactionID) (ret ledgerstate.InclusionState, found bool) {
	found = false
	messagelayer.Tangle().LedgerState.Transaction(txid).Consume(func(tx *ledgerstate.Transaction) {
		branchID := messagelayer.Tangle().LedgerState.BranchID(txid)
		found = true
		ret = messagelayer.Tangle().LedgerState.BranchInclusionState(branchID)
	})
	return
}

func (v *ValueTangle) PostTransaction(tx *ledgerstate.Transaction) error {
	_, err := issuer.IssuePayload(tx, messagelayer.Tangle())
	if err != nil {
		return fmt.Errorf("failed to issue transaction: %w", err)
	}
	return nil
}

func (v *ValueTangle) RequestFunds(target ledgerstate.Address) error {
	faucetPayload, err := faucet.NewRequest(target, config.Node().Int(faucet.CfgFaucetPoWDifficulty))
	if err != nil {
		return err
	}
	_, err = messagelayer.Tangle().MessageFactory.IssuePayload(faucetPayload, messagelayer.Tangle())
	return err
}
