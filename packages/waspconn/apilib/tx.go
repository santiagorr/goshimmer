package apilib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/transaction"
	"github.com/mr-tron/base58"
)

type PostTransactionRequest struct {
	Tx string `json:"tx"` // Transaction.Bytes() encoded as base58
}

type PostTransactionResponse struct {
	Err string
}

func PostTransaction(netLoc string, tx *transaction.Transaction) error {
	req := &PostTransactionRequest{Tx: base58.Encode(tx.Bytes())}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("http://%s/utxodb/tx", netLoc)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	res := &PostTransactionResponse{}
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK || res.Err != "" {
		return fmt.Errorf("/utxodb/tx returned code %d: %s", resp.StatusCode, res.Err)
	}
	return nil
}
