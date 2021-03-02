// +build ignorepackage apilib

import (
	"encoding/json"
	"fmt"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/transaction"
	"net/http"
)

type IsConfirmedResponse struct {
	Confirmed bool   `json:"confirmed"`
	Err       string `json:"err"`
}

func IsConfirmed(netLoc string, txid *transaction.ID) (bool, error) {
	url := fmt.Sprintf("http://%s/utxodb/confirmed/%s", netLoc, txid.String())
	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	res := &IsConfirmedResponse{}
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK || res.Err != "" {
		return false, fmt.Errorf("%s returned code %d: %s", url, resp.StatusCode, res.Err)
	}
	return res.Confirmed, nil
}
