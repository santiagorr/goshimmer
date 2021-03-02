// +build ignorepackage apilib

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/transaction"
)

type OutputBalance struct {
	Value int64  `json:"value"`
	Color string `json:"color"` // base58
}

type GetAccountOutputsResponse struct {
	Address string                     `json:"address"` // base58
	Outputs map[string][]OutputBalance `json:"outputs"` // map[output id as base58]balance
	Err     string                     `json:"err"`
}

func GetAccountOutputs(netLoc string, address *address.Address) (map[transaction.OutputID][]*balance.Balance, error) {
	url := fmt.Sprintf("http://%s/utxodb/outputs/%s", netLoc, address.String())
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	res := &GetAccountOutputsResponse{}
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK || res.Err != "" {
		return nil, fmt.Errorf("%s returned code %d: %s", url, resp.StatusCode, res.Err)
	}

	outputs := make(map[transaction.OutputID][]*balance.Balance)
	for k, v := range res.Outputs {
		id, err := transaction.OutputIDFromBase58(k)
		if err != nil {
			return nil, err
		}
		balances := make([]*balance.Balance, len(v))
		for i, b := range v {
			color, err := transaction.IDFromBase58(b.Color)
			if err != nil {
				return nil, err
			}
			balances[i] = balance.New(balance.Color(color), b.Value)
		}
		outputs[id] = balances
	}
	return outputs, nil
}
