// +build ignorepackage connector

import (
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/transaction"
	"github.com/iotaledger/hive.go/events"
)

// EventValueTransactionConfirmed global event.
// Triggered whenever new confirmed transaction is confirmed
var (
	EventValueTransactionConfirmed *events.Event
	EventValueTransactionBooked    *events.Event
	EventValueTransactionRejected  *events.Event
)

func init() {
	EventValueTransactionConfirmed = events.NewEvent(func(handler interface{}, params ...interface{}) {
		handler.(func(_ *transaction.Transaction))(params[0].(*transaction.Transaction))
	})
	EventValueTransactionBooked = events.NewEvent(func(handler interface{}, params ...interface{}) {
		handler.(func(_ *transaction.Transaction))(params[0].(*transaction.Transaction))
	})
	EventValueTransactionRejected = events.NewEvent(func(handler interface{}, params ...interface{}) {
		handler.(func(_ *transaction.Transaction))(params[0].(*transaction.Transaction))
	})
}
