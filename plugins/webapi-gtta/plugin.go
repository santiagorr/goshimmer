package webapi_gtta

import (
	"net/http"
	"time"

	"github.com/iotaledger/goshimmer/packages/node"
	"github.com/iotaledger/goshimmer/packages/ternary"
	"github.com/iotaledger/goshimmer/plugins/tipselection"
	"github.com/iotaledger/goshimmer/plugins/webapi"
	"github.com/labstack/echo"
)

var PLUGIN = node.NewPlugin("WebAPI GTTA Endpoint", func(plugin *node.Plugin) {
	webapi.AddEndpoint("getTransactionsToApprove", Handler)
})

func Handler(c echo.Context) error {
	start := time.Now()

	branchTransactionHash := tipselection.GetRandomTip()
	trunkTransactionHash := tipselection.GetRandomTip()

	return c.JSON(http.StatusOK, response{
		Duration:          time.Since(start).Nanoseconds() / 1e6,
		BranchTransaction: branchTransactionHash,
		TrunkTransaction:  trunkTransactionHash,
	})
}

type response struct {
	Duration          int64           `json:"duration"`
	BranchTransaction ternary.Trinary `json:"branchTransaction"`
	TrunkTransaction  ternary.Trinary `json:"trunkTransaction"`
}