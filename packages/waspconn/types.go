package waspconn

import (
	"github.com/iotaledger/goshimmer/packages/ledgerstate"
)

// ValueTangle is the interface between waspconn and the underlying value tangle
type ValueTangle interface {
	GetAddressOutputs(addr ledgerstate.Address) map[ledgerstate.OutputID]*ledgerstate.ColoredBalances
	GetTransaction(txid ledgerstate.TransactionID, f func(*ledgerstate.Transaction)) bool
	GetBranchInclusionState(txid ledgerstate.TransactionID) (ledgerstate.InclusionState, bool)
	OnTransactionConfirmed(func(tx *ledgerstate.Transaction))
	OnTransactionBooked(func(tx *ledgerstate.Transaction))
	PostTransaction(tx *ledgerstate.Transaction) error
	RequestFunds(target ledgerstate.Address) error
	Detach()
}
