package testing

import (
	"github.com/iotaledger/goshimmer/packages/waspconn/valuetangle"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/node"
)

var log *logger.Logger

func Config(_ *node.Plugin, setLog *logger.Logger, vtangle valuetangle.ValueTangle) {
	log = setLog.Named("testing")
	addEndpoints(vtangle)
}
