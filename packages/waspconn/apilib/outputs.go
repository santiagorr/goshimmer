package apilib

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/iotaledger/goshimmer/packages/ledgerstate"
)

type OutputBalance struct {
	Value uint64 `json:"value"`
	Color string `json:"color"` // base58
}

type GetAccountOutputsResponse struct {
	Address string                     `json:"address"` // base58
	Outputs map[string][]OutputBalance `json:"outputs"` // map[output id as base58]balance
	Err     string                     `json:"err"`
}

func GetAccountOutputs(netLoc string, address ledgerstate.Address) (map[ledgerstate.OutputID]*ledgerstate.ColoredBalances, error) {
	url := fmt.Sprintf("http://%s/utxodb/outputs/%s", netLoc, address.Base58())
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

	outputs := make(map[ledgerstate.OutputID]*ledgerstate.ColoredBalances)
	for k, v := range res.Outputs {
		id, err := ledgerstate.OutputIDFromBase58(k)
		if err != nil {
			return nil, err
		}
		balances := make(map[ledgerstate.Color]uint64)
		for _, b := range v {
			color, err := ledgerstate.ColorFromBase58EncodedString(b.Color)
			if err != nil {
				return nil, err
			}
			balances[color] = b.Value
		}
		outputs[id] = ledgerstate.NewColoredBalances(balances)
	}
	return outputs, nil
}
