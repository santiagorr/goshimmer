package connector

import (
	"io"
	"net"
	"strings"

	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"github.com/iotaledger/goshimmer/packages/shutdown"
	"github.com/iotaledger/goshimmer/packages/tangle"
	"github.com/iotaledger/goshimmer/packages/waspconn"
	"github.com/iotaledger/goshimmer/packages/waspconn/chopper"
	"github.com/iotaledger/hive.go/daemon"
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/netutil/buffconn"
)

type WaspConnector struct {
	id                                 string
	bconn                              *buffconn.BufferedConnection
	subscriptions                      map[ledgerstate.Address]ledgerstate.Color
	inTxChan                           chan interface{}
	exitConnChan                       chan struct{}
	receiveConfirmedTransactionClosure *events.Closure
	receiveBookedTransactionClosure    *events.Closure
	receiveWaspMessageClosure          *events.Closure
	messageChopper                     *chopper.Chopper
	log                                *logger.Logger
	vtangle                            waspconn.ValueTangle
}

type wrapConfirmedTx *ledgerstate.Transaction
type wrapBookedTx *ledgerstate.Transaction

func Run(conn net.Conn, log *logger.Logger, vtangle waspconn.ValueTangle) {
	wconn := &WaspConnector{
		bconn:          buffconn.NewBufferedConnection(conn, tangle.MaxMessageSize),
		exitConnChan:   make(chan struct{}),
		messageChopper: chopper.NewChopper(),
		log:            log,
		vtangle:        vtangle,
	}
	err := daemon.BackgroundWorker(wconn.Id(), func(shutdownSignal <-chan struct{}) {
		select {
		case <-shutdownSignal:
			wconn.log.Infof("shutdown signal received..")
			_ = wconn.bconn.Close()

		case <-wconn.exitConnChan:
			wconn.log.Infof("closing connection..")
			_ = wconn.bconn.Close()
		}

		go wconn.detach()
	}, shutdown.PriorityTangle) // TODO proper priority

	if err != nil {
		close(wconn.exitConnChan)
		wconn.log.Errorf("can't start deamon")
		return
	}
	wconn.attach()
}

func (wconn *WaspConnector) Id() string {
	if wconn.id == "" {
		return "wasp_" + wconn.bconn.RemoteAddr().String()
	}
	return wconn.id
}

func (wconn *WaspConnector) SetId(id string) {
	wconn.id = id
	wconn.log = wconn.log.Named(id)
	wconn.log.Infof("wasp connection id has been set to '%s' for '%s'", id, wconn.bconn.RemoteAddr().String())
}

func (wconn *WaspConnector) attach() {
	wconn.subscriptions = make(map[ledgerstate.Address]ledgerstate.Color)
	wconn.inTxChan = make(chan interface{})

	wconn.receiveConfirmedTransactionClosure = events.NewClosure(func(vtx *ledgerstate.Transaction) {
		wconn.inTxChan <- wrapConfirmedTx(vtx)
	})

	wconn.receiveBookedTransactionClosure = events.NewClosure(func(vtx *ledgerstate.Transaction) {
		wconn.inTxChan <- wrapBookedTx(vtx)
	})

	wconn.receiveWaspMessageClosure = events.NewClosure(func(data []byte) {
		wconn.processMsgDataFromWasp(data)
	})

	// attach connector to the flow of incoming value transactions
	EventValueTransactionConfirmed.Attach(wconn.receiveConfirmedTransactionClosure)
	EventValueTransactionBooked.Attach(wconn.receiveBookedTransactionClosure)

	wconn.bconn.Events.ReceiveMessage.Attach(wconn.receiveWaspMessageClosure)

	wconn.log.Debugf("attached waspconn")

	// read connection thread
	go func() {
		if err := wconn.bconn.Read(); err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
				wconn.log.Warnw("Permanent error", "err", err)
			}
		}
		close(wconn.exitConnChan)
	}()

	// read incoming pre-filtered transactions from node
	go func() {
		for vtx := range wconn.inTxChan {
			switch tvtx := vtx.(type) {
			case wrapConfirmedTx:
				wconn.processConfirmedTransactionFromNode(tvtx)

			case wrapBookedTx:
				wconn.processBookedTransactionFromNode(tvtx)

			default:
				wconn.log.Panicf("wrong type")
			}
		}
	}()
}

func (wconn *WaspConnector) detach() {
	EventValueTransactionConfirmed.Detach(wconn.receiveConfirmedTransactionClosure)
	EventValueTransactionBooked.Detach(wconn.receiveBookedTransactionClosure)
	wconn.bconn.Events.ReceiveMessage.Detach(wconn.receiveWaspMessageClosure)

	wconn.messageChopper.Close()
	close(wconn.inTxChan)
	_ = wconn.bconn.Close()
	wconn.log.Debugf("detached waspconn")
}

func (wconn *WaspConnector) subscribe(addr ledgerstate.Address, color ledgerstate.Color) {
	_, ok := wconn.subscriptions[addr]
	if !ok {
		wconn.log.Infof("subscribed to address %s with color %s", addr.String(), color.String())
		wconn.subscriptions[addr] = color
	}
}

func (wconn *WaspConnector) isSubscribed(addr ledgerstate.Address) bool {
	_, ok := wconn.subscriptions[addr]
	return ok
}

func (wconn *WaspConnector) txSubscribedAddresses(tx *ledgerstate.Transaction) []ledgerstate.Address {
	ret := make([]ledgerstate.Address, 0)
	for _, output := range tx.Essence().Outputs() {
		addr := output.Address()
		if wconn.isSubscribed(addr) {
			ret = append(ret, addr)
		}
	}
	return ret
}

// processConfirmedTransactionFromNode receives only confirmed transactions
// it parses SC transaction incoming from the node. Forwards it to Wasp if subscribed
func (wconn *WaspConnector) processConfirmedTransactionFromNode(tx *ledgerstate.Transaction) {
	// determine if transaction contains any of subscribed addresses in its outputs
	wconn.log.Debugw("processConfirmedTransactionFromNode", "txid", tx.ID().String())

	subscribedOutAddresses := wconn.txSubscribedAddresses(tx)
	if len(subscribedOutAddresses) == 0 {
		wconn.log.Debugw("not subscribed", "txid", tx.ID().String())
		// dismiss unsubscribed transaction
		return
	}
	// for each subscribed address retrieve outputs and send to wasp with the transaction
	wconn.log.Debugf("txid %s contains %d subscribed addresses", tx.ID().String(), len(subscribedOutAddresses))

	for i := range subscribedOutAddresses {
		outs := wconn.vtangle.GetAddressOutputs(subscribedOutAddresses[i])
		bals := waspconn.OutputsToBalances(outs)
		err := wconn.sendAddressUpdateToWasp(
			subscribedOutAddresses[i],
			bals,
			tx,
		)
		if err != nil {
			wconn.log.Errorf("sendAddressUpdateToWasp: %v", err)
		} else {
			wconn.log.Infof("confirmed tx -> Wasp: sc addr: %s, txid: %s",
				subscribedOutAddresses[i].String(), tx.ID().String())
		}
	}
}

func (wconn *WaspConnector) processBookedTransactionFromNode(tx *ledgerstate.Transaction) {
	addrs := wconn.txSubscribedAddresses(tx)
	if len(addrs) == 0 {
		return
	}
	txid := tx.ID()
	if err := wconn.sendBranchInclusionStateToWasp(ledgerstate.Pending, txid, addrs); err != nil {
		wconn.log.Errorf("processBookedTransactionFromNode: %v", err)
	} else {
		wconn.log.Infof("booked tx -> Wasp. txid: %s", tx.ID().String())
	}
}

func (wconn *WaspConnector) getConfirmedTransaction(txid ledgerstate.TransactionID) {
	wconn.log.Debugf("requested transaction id = %s", txid.String())

	found := wconn.vtangle.GetTransaction(txid, func(tx *ledgerstate.Transaction) {
		state, _ := wconn.vtangle.GetBranchInclusionState(txid)
		if state != ledgerstate.Confirmed {
			wconn.log.Warnf("GetConfirmedTransaction: not confirmed %s", txid.String())
			return
		}
		if err := wconn.sendConfirmedTransactionToWasp(tx); err != nil {
			wconn.log.Errorf("sendConfirmedTransactionToWasp: %v", err)
			return
		}
		wconn.log.Infof("confirmed tx -> Wasp. txid = %s", txid.String())
	})
	if !found {
		wconn.log.Warnf("GetConfirmedTransaction: not found %s", txid.String())
		return
	}
}

func (wconn *WaspConnector) getBranchInclusionState(txid ledgerstate.TransactionID, addr ledgerstate.Address) {
	state, found := wconn.vtangle.GetBranchInclusionState(txid)
	if !found {
		return
	}
	if err := wconn.sendBranchInclusionStateToWasp(state, txid, []ledgerstate.Address{addr}); err != nil {
		wconn.log.Errorf("sendBranchInclusionStateToWasp: %v", err)
		return
	}
}

func (wconn *WaspConnector) getAddressBalance(addr ledgerstate.Address) {
	wconn.log.Debugf("getAddressBalance request for address: %s", addr.String())

	outputs := wconn.vtangle.GetAddressOutputs(addr)
	if len(outputs) == 0 {
		return
	}
	ret := waspconn.OutputsToBalances(outputs)

	wconn.log.Debugf("sending balances to wasp: %s    %+v", addr.String(), ret)

	if err := wconn.sendAddressOutputsToWasp(addr, ret); err != nil {
		wconn.log.Debugf("sendAddressOutputsToWasp: %v", err)
	}
}

func (wconn *WaspConnector) postTransaction(tx *ledgerstate.Transaction, fromSC ledgerstate.Address, fromLeader uint16) {
	if err := wconn.vtangle.PostTransaction(tx); err != nil {
		wconn.log.Warnf("%v: %s", err, tx.ID().String())
		return
	}
	wconn.log.Infof("Wasp -> Tangle. txid: %s, from sc: %s, from leader: %d",
		tx.ID().String(), fromSC.String(), fromLeader)
}
