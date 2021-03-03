package connector

import (
	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"github.com/iotaledger/hive.go/events"
)

// EventValueTransactionConfirmed global event.
// Triggered whenever new confirmed transaction is confirmed
var (
	EventValueTransactionConfirmed *events.Event
	EventValueTransactionBooked    *events.Event
)

func init() {
	EventValueTransactionConfirmed = events.NewEvent(func(handler interface{}, params ...interface{}) {
		handler.(func(_ *ledgerstate.Transaction))(params[0].(*ledgerstate.Transaction))
	})
	EventValueTransactionBooked = events.NewEvent(func(handler interface{}, params ...interface{}) {
		handler.(func(_ *ledgerstate.Transaction))(params[0].(*ledgerstate.Transaction))
	})
}
