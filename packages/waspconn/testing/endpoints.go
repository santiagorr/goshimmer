// +build ignorepackage testing

import (
	"fmt"
	"net/http"

	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/transaction"
	"github.com/iotaledger/goshimmer/packages/waspconn/apilib"
	"github.com/iotaledger/goshimmer/packages/waspconn/valuetangle"
	"github.com/iotaledger/goshimmer/plugins/gracefulshutdown"
	"github.com/iotaledger/goshimmer/plugins/webapi"
	"github.com/labstack/echo"
	"github.com/mr-tron/base58"
)

func addEndpoints(vtangle valuetangle.ValueTangle) {
	t := &testingHandler{vtangle}

	webapi.Server().GET("/utxodb/outputs/:address", t.handleGetAddressOutputs)
	webapi.Server().GET("/utxodb/confirmed/:txid", t.handleIsConfirmed)
	webapi.Server().POST("/utxodb/tx", t.handlePostTransaction)
	webapi.Server().GET("/utxodb/requestfunds/:address", t.handleRequestFunds)
	webapi.Server().GET("/adm/shutdown", handleShutdown)

	log.Info("addded UTXODB endpoints")
}

type testingHandler struct {
	vtangle valuetangle.ValueTangle
}

func (t *testingHandler) handleGetAddressOutputs(c echo.Context) error {
	log.Debugw("handleGetAddressOutputs")
	addr, err := address.FromBase58(c.Param("address"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, &apilib.GetAccountOutputsResponse{Err: err.Error()})
	}
	outputs, err := t.vtangle.GetConfirmedAddressOutputs(addr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, &apilib.GetAccountOutputsResponse{Err: err.Error()})
	}
	log.Debugf("handleGetAddressOutputs: addr %s from utxodb %+v", addr.String(), outputs)

	out := make(map[string][]apilib.OutputBalance)
	for txOutId, txOutputs := range outputs {
		txOut := make([]apilib.OutputBalance, len(txOutputs))
		for i, txOutput := range txOutputs {
			txOut[i] = apilib.OutputBalance{
				Value: txOutput.Value,
				Color: transaction.ID(txOutput.Color).String(),
			}
		}
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

	tx, _, err := transaction.FromBytes(txBytes)
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

func (t *testingHandler) handleIsConfirmed(c echo.Context) error {
	log.Debugw("handleIsConfirmed")
	txid, err := transaction.IDFromBase58(c.Param("txid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, &apilib.IsConfirmedResponse{Err: err.Error()})
	}
	confirmed, err := t.vtangle.IsConfirmed(&txid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, &apilib.IsConfirmedResponse{Err: err.Error()})
	}
	log.Debugf("handleIsConfirmed: txid %s confirmed = %v", txid.String(), confirmed)

	return c.JSON(http.StatusOK, &apilib.IsConfirmedResponse{Confirmed: confirmed})
}

func (t *testingHandler) handleRequestFunds(c echo.Context) error {
	addr, err := address.FromBase58(c.Param("address"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, &apilib.RequestFundsResponse{Err: err.Error()})
	}
	err = t.vtangle.RequestFunds(addr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, &apilib.RequestFundsResponse{Err: err.Error()})
	}
	return c.JSON(http.StatusOK, &apilib.RequestFundsResponse{})
}
