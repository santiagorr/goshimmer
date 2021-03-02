package waspconn

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/transaction"
	"github.com/iotaledger/goshimmer/packages/waspconn/connector"
	"github.com/iotaledger/goshimmer/packages/waspconn/testing"
	"github.com/iotaledger/goshimmer/packages/waspconn/utxodb"
	"github.com/iotaledger/goshimmer/packages/waspconn/valuetangle"
	"github.com/iotaledger/goshimmer/packages/shutdown"
	"github.com/iotaledger/goshimmer/plugins/config"
	"github.com/iotaledger/hive.go/daemon"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/node"
	flag "github.com/spf13/pflag"
)

const (
	PluginName = "WaspConn"

	WaspConnPort          = "waspconn.port"
	WaspConnUtxodbEnabled = "waspconn.utxodbenabled"

	WaspConnUtxodbConfirmDelay           = "waspconn.utxodbconfirmseconds"
	WaspConnUtxodbConfirmRandomize       = "waspconn.utxodbconfirmrandomize"
	WaspConnUtxodbConfirmFirstInConflict = "waspconn.utxodbconfirmfirst"
)

func init() {
	flag.Int(WaspConnPort, 5000, "port for Wasp connections")
	flag.Bool(WaspConnUtxodbEnabled, false, "is utxodb mocking the value tangle enabled")

	flag.Int(WaspConnUtxodbConfirmDelay, 0, "emulated confirmation delay for utxodb in seconds")
	flag.Bool(WaspConnUtxodbConfirmRandomize, false, "is confirmation time random with the mean at confirmation delay")
	flag.Bool(WaspConnUtxodbConfirmFirstInConflict, false, "in case of conflict, confirm first transaction. Default is reject all")
}

var (
	app     *node.Plugin
	appOnce sync.Once

	PLUGINS = node.Plugins(
		App(),
	)
	log *logger.Logger

	vtangle valuetangle.ValueTangle
)

func App() *node.Plugin {
	appOnce.Do(func() {
		app = node.NewPlugin(PluginName, node.Enabled, configPlugin, runPlugin)
	})
	return app
}

func configPlugin(plugin *node.Plugin) {
	log = logger.NewLogger(PluginName)

	utxodbEnabled := config.Node().Bool(WaspConnUtxodbEnabled)

	if utxodbEnabled {
		confirmTime := time.Duration(config.Node().Int(WaspConnUtxodbConfirmDelay)) * time.Second
		randomize := config.Node().Bool(WaspConnUtxodbConfirmRandomize)
		confirmFirstInConflict := config.Node().Bool(WaspConnUtxodbConfirmFirstInConflict)

		vtangle = utxodb.NewConfirmEmulator(confirmTime, randomize, confirmFirstInConflict)
		testing.Config(plugin, log, vtangle)
		log.Infof("configured with UTXODB enabled")
	} else {
		vtangle = valuetangle.NewRealValueTangle()
		log.Infof("configured for ValueTangle")
	}
	vtangle.OnTransactionConfirmed(func(tx *transaction.Transaction) {
		log.Debugf("on transaction confirmed: %s", tx.ID().String())
		connector.EventValueTransactionConfirmed.Trigger(tx)
	})
	vtangle.OnTransactionBooked(func(tx *transaction.Transaction, decisionPending bool) {
		log.Debugf("on transaction booked: %s, decisionPending: %v", tx.ID().String(), decisionPending)
		connector.EventValueTransactionBooked.Trigger(tx)
	})
	vtangle.OnTransactionRejected(func(tx *transaction.Transaction) {
		log.Debugf("on transaction rejected: %s", tx.ID().String())
		connector.EventValueTransactionRejected.Trigger(tx)
	})
}

func runPlugin(_ *node.Plugin) {
	log.Debugf("starting WaspConn plugin on port %d", config.Node().Int(WaspConnPort))
	port := config.Node().Int(WaspConnPort)
	err := daemon.BackgroundWorker("WaspConn worker", func(shutdownSignal <-chan struct{}) {
		listenOn := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", listenOn)
		if err != nil {
			log.Errorf("failed to start WaspConn daemon: %v", err)
			return
		}
		defer func() {
			_ = listener.Close()
		}()

		go func() {
			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				log.Debugf("accepted connection from %s", conn.RemoteAddr().String())
				connector.Run(conn, log, vtangle)
			}
		}()

		log.Debugf("running WaspConn plugin on port %d", port)

		<-shutdownSignal

		log.Infof("Detaching WaspConn from the Value Tangle..")
		go func() {
			vtangle.Detach()
			log.Infof("Detaching WaspConn from the Value Tangle..Done")
		}()

	}, shutdown.PriorityValueTangle) // TODO proper shutdown priority
	if err != nil {
		log.Errorf("failed to start WaspConn daemon: %v", err)
	}
}
