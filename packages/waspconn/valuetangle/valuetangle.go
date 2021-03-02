package valuetangle

import (
	"fmt"

	"github.com/iotaledger/goshimmer/packages/waspconn"

	"github.com/iotaledger/goshimmer/packages/valuetransfers"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/tangle"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/transaction"
	"github.com/iotaledger/goshimmer/plugins/config"
	"github.com/iotaledger/goshimmer/plugins/faucet"
	"github.com/iotaledger/goshimmer/plugins/issuer"
	"github.com/iotaledger/goshimmer/plugins/messagelayer"
	"github.com/iotaledger/hive.go/events"
)

// ValueTangle is the interface between waspconn and the value tangle
type ValueTangle interface {
	GetConfirmedAddressOutputs(addr address.Address) (map[transaction.OutputID][]*balance.Balance, error)
	GetConfirmedTransaction(txid *transaction.ID) *transaction.Transaction
	GetTxInclusionLevel(txid *transaction.ID) byte
	OnTransactionConfirmed(func(tx *transaction.Transaction))
	OnTransactionBooked(func(tx *transaction.Transaction, decisionPending bool))
	OnTransactionRejected(func(tx *transaction.Transaction))
	IsConfirmed(txid *transaction.ID) (bool, error)
	PostTransaction(tx *transaction.Transaction) error
	RequestFunds(target address.Address) error
	Detach()
}

type valuetangle struct {
	txConfirmedClosure  *events.Closure
	txConfirmedCallback func(tx *transaction.Transaction)

	txBookedClosure  *events.Closure
	txBookedCallback func(tx *transaction.Transaction, decisionPending bool)

	txRejectedClosure  *events.Closure
	txRejectedCallback func(tx *transaction.Transaction)
}

// NewRealValueTangle returns an implementation for ValueTangle
func NewRealValueTangle() *valuetangle {
	v := &valuetangle{}

	v.txConfirmedClosure = events.NewClosure(func(e *tangle.CachedTransactionEvent) {
		defer e.Transaction.Release()
		defer e.TransactionMetadata.Release()

		if v.txConfirmedCallback == nil {
			return
		}
		if tx := e.Transaction.Unwrap(); tx != nil {
			v.txConfirmedCallback(tx)
		}
	})
	valuetransfers.Tangle().Events.TransactionConfirmed.Attach(v.txConfirmedClosure)

	v.txBookedClosure = events.NewClosure(func(e *tangle.CachedTransactionBookEvent) {
		defer e.Transaction.Release()
		defer e.TransactionMetadata.Release()

		if v.txBookedCallback == nil {
			return
		}
		if tx := e.Transaction.Unwrap(); tx != nil {
			v.txBookedCallback(tx, e.Pending)
		}
	})
	valuetransfers.Tangle().Events.TransactionBooked.Attach(v.txBookedClosure)

	v.txRejectedClosure = events.NewClosure(func(e *tangle.CachedTransactionEvent) {
		defer e.Transaction.Release()
		defer e.TransactionMetadata.Release()

		if v.txRejectedCallback == nil {
			return
		}
		if tx := e.Transaction.Unwrap(); tx != nil {
			v.txRejectedCallback(tx)
		}
	})
	valuetransfers.Tangle().Events.TransactionRejected.Attach(v.txRejectedClosure)

	return v
}

func (v *valuetangle) Detach() {
	valuetransfers.Tangle().Events.TransactionConfirmed.Detach(v.txConfirmedClosure)
	valuetransfers.Tangle().Events.TransactionBooked.Detach(v.txBookedClosure)
	valuetransfers.Tangle().Events.TransactionRejected.Detach(v.txRejectedClosure)
}

func (v *valuetangle) OnTransactionConfirmed(cb func(tx *transaction.Transaction)) {
	v.txConfirmedCallback = cb
}

func (v *valuetangle) OnTransactionBooked(cb func(tx *transaction.Transaction, decisionPending bool)) {
	v.txBookedCallback = cb
}

func (v *valuetangle) OnTransactionRejected(cb func(tx *transaction.Transaction)) {
	v.txRejectedCallback = cb
}

// GetConfirmedAddressOutputs return confirmed UTXOs for address
func (v *valuetangle) GetConfirmedAddressOutputs(addr address.Address) (map[transaction.OutputID][]*balance.Balance, error) {
	ret := make(map[transaction.OutputID][]*balance.Balance)
	valuetransfers.Tangle().OutputsOnAddress(addr).Consume(func(output *tangle.Output) {
		if output.Confirmed() && output.ConsumerCount() == 0 {
			ret[output.ID()] = output.Balances()
		}
	})
	return ret, nil
}

// GetConfirmedTransaction returns transaction and its simplified inclusion state, the confirmation flag
// if transaction does not exist of it is rejected, return (nil, false)
func (v *valuetangle) GetConfirmedTransaction(txid *transaction.ID) *transaction.Transaction {
	// retrieve transaction
	cachedTxnObj := valuetransfers.Tangle().Transaction(*txid)
	defer cachedTxnObj.Release()

	if !cachedTxnObj.Exists() {
		return nil
	}

	// retrieve metadata
	cachedTxnMetaObj := valuetransfers.Tangle().TransactionMetadata(*txid)
	defer cachedTxnMetaObj.Release()

	if !cachedTxnMetaObj.Exists() {
		return nil
	}
	if !cachedTxnMetaObj.Unwrap().Confirmed() {
		return nil
	}
	return cachedTxnObj.Unwrap()
}

func (ce *valuetangle) GetTxInclusionLevel(txid *transaction.ID) byte {
	cachedTxnObj := valuetransfers.Tangle().Transaction(*txid)
	defer cachedTxnObj.Release()

	if !cachedTxnObj.Exists() {
		return waspconn.TransactionInclusionLevelUndef
	}

	// retrieve metadata
	cachedTxnMetaObj := valuetransfers.Tangle().TransactionMetadata(*txid)
	defer cachedTxnMetaObj.Release()

	unwrapped := cachedTxnMetaObj.Unwrap()
	switch {
	case !cachedTxnMetaObj.Exists():
		return waspconn.TransactionInclusionLevelUndef

	case unwrapped.Rejected():
		return waspconn.TransactionInclusionLevelRejected

	case unwrapped.Confirmed():
		return waspconn.TransactionInclusionLevelConfirmed

	default:
		return waspconn.TransactionInclusionLevelBooked
	}
}

func (v *valuetangle) PostTransaction(tx *transaction.Transaction) error {
	// prepare value payload with value factory
	payload, err := valuetransfers.ValueObjectFactory().IssueTransaction(tx)
	if err != nil {
		return fmt.Errorf("failed to issue transaction: %w", err)
	}

	// attach to message layer
	_, err = issuer.IssuePayload(payload)
	return err
}

func (v *valuetangle) IsConfirmed(txid *transaction.ID) (bool, error) {
	cachedTxnMetaObj := valuetransfers.Tangle().TransactionMetadata(*txid)
	defer cachedTxnMetaObj.Release()
	if !cachedTxnMetaObj.Exists() {
		return false, fmt.Errorf("Transaction not found")
	}
	return cachedTxnMetaObj.Unwrap().Confirmed(), nil
}

func (v *valuetangle) RequestFunds(target address.Address) error {
	faucetPayload, err := faucet.NewRequest(target, config.Node().Int(faucet.CfgFaucetPoWDifficulty))
	if err != nil {
		return err
	}
	_, err = messagelayer.Tangle().MessageFactory.IssuePayload(faucetPayload)
	return err
}
