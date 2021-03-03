package testing

import (
	"fmt"
	"net/http"

	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"github.com/iotaledger/goshimmer/packages/waspconn"
	"github.com/iotaledger/goshimmer/packages/waspconn/apilib"
	"github.com/iotaledger/goshimmer/plugins/gracefulshutdown"
	"github.com/iotaledger/goshimmer/plugins/webapi"
	"github.com/labstack/echo"
	"github.com/mr-tron/base58"
)

func addEndpoints(vtangle waspconn.ValueTangle) {
	t := &testingHandler{vtangle}

	webapi.Server().GET("/utxodb/outputs/:address", t.handleGetAddressOutputs)
	webapi.Server().GET("/utxodb/inclusionstate/:txid", t.handleInclusionState)
	webapi.Server().POST("/utxodb/tx", t.handlePostTransaction)
	webapi.Server().GET("/utxodb/requestfunds/:address", t.handleRequestFunds)
	webapi.Server().GET("/adm/shutdown", handleShutdown)

	log.Info("addded UTXODB endpoints")
}

type testingHandler struct {
	vtangle waspconn.ValueTangle
}

func (t *testingHandler) handleGetAddressOutputs(c echo.Context) error {
	log.Debugw("handleGetAddressOutputs")
	addr, err := ledgerstate.AddressFromBase58EncodedString(c.Param("address"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, &apilib.GetAccountOutputsResponse{Err: err.Error()})
	}
	outputs := t.vtangle.GetAddressOutputs(addr)
	log.Debugf("handleGetAddressOutputs: addr %s from utxodb %+v", addr.String(), outputs)

	out := make(map[string][]apilib.OutputBalance)
	for txOutId, txOutputs := range outputs {
		txOut := make([]apilib.OutputBalance, 0)
		txOutputs.ForEach(func(color ledgerstate.Color, balance uint64) bool {
			txOut = append(txOut, apilib.OutputBalance{
				Value: balance,
				Color: color.Base58(),
			})
			return true
		})
		out[txOutId.String()] = txOut
	}
	log.Debugw("handleGetAddressOutputs", "sending", out)

	return c.JSONPretty(http.StatusOK, &apilib.GetAccountOutputsResponse{
		Address: c.Param("address"),
		Outputs: out,
	}, " ")
}

func (t *testingHandler) handlePostTransaction(c echo.Context) error {
	var req apilib.PostTransactionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, &apilib.PostTransactionResponse{Err: err.Error()})
	}

	txBytes, err := base58.Decode(req.Tx)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &apilib.PostTransactionResponse{Err: err.Error()})
	}

	tx, _, err := ledgerstate.TransactionFromBytes(txBytes)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &apilib.PostTransactionResponse{Err: err.Error()})
	}

	log.Debugf("handlePostTransaction:utxodb.AddTransaction: txid %s", tx.ID().String())

	err = t.vtangle.PostTransaction(tx)
	if err != nil {
		log.Warnf("handlePostTransaction:utxodb.AddTransaction: txid %s err = %v", tx.ID().String(), err)
		return c.JSON(http.StatusConflict, &apilib.PostTransactionResponse{Err: err.Error()})
	}

	return c.JSON(http.StatusOK, &apilib.PostTransactionResponse{})
}

func handleShutdown(c echo.Context) error {
	gracefulshutdown.ShutdownWithError(fmt.Errorf("Shutdown requested from WebAPI."))
	return nil
}

func (t *testingHandler) handleInclusionState(c echo.Context) error {
	log.Debugw("handleInclusionState")
	txid, err := ledgerstate.TransactionIDFromBase58(c.Param("txid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, &apilib.InclusionStateResponse{Err: err.Error()})
	}
	state, found := t.vtangle.GetBranchInclusionState(txid)
	if !found {
		return c.JSON(http.StatusNotFound, &apilib.InclusionStateResponse{Err: "Not found"})
	}
	log.Debugf("handleInclusionState: txid %s state = %v", txid.String(), state)

	return c.JSON(http.StatusOK, &apilib.InclusionStateResponse{State: state})
}

func (t *testingHandler) handleRequestFunds(c echo.Context) error {
	addr, err := ledgerstate.AddressFromBase58EncodedString(c.Param("address"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, &apilib.RequestFundsResponse{Err: err.Error()})
	}
	err = t.vtangle.RequestFunds(addr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, &apilib.RequestFundsResponse{Err: err.Error()})
	}
	return c.JSON(http.StatusOK, &apilib.RequestFundsResponse{})
}
