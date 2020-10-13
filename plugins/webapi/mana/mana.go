package mana

import (
	"net/http"

	manaPkg "github.com/iotaledger/goshimmer/packages/mana"
	"github.com/iotaledger/goshimmer/plugins/autopeering/local"
	manaPlugin "github.com/iotaledger/goshimmer/plugins/mana"
	"github.com/iotaledger/hive.go/identity"
	"github.com/labstack/echo"
	"github.com/mr-tron/base58"
)

// getManaHandler handles the request.
func getManaHandler(c echo.Context) error {
	var request GetManaRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, GetManaResponse{Error: err.Error()})
	}
	ID, err := manaPkg.IDFromStr(request.Node)
	if err != nil {
		return c.JSON(http.StatusBadRequest, GetManaResponse{Error: err.Error()})
	}
	emptyID := identity.ID{}
	if ID == emptyID {
		ID = local.GetInstance().ID()
	}
	accessMana, err := manaPlugin.GetAccessMana(ID, manaPkg.Mixed)
	if err != nil {
		return c.JSON(http.StatusBadRequest, GetManaResponse{Error: err.Error()})
	}
	consensusMana, err := manaPlugin.GetConsensusMana(ID, manaPkg.Mixed)
	if err != nil {
		return c.JSON(http.StatusBadRequest, GetManaResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, GetManaResponse{
		Node:      base58.Encode(ID.Bytes()),
		Access:    accessMana,
		Consensus: consensusMana,
	})
}

// GetManaRequest is the request for get mana.
type GetManaRequest struct {
	Node string `json:"node"`
}

// GetManaResponse defines the response for get mana.
type GetManaResponse struct {
	Error     string  `json:"error,omitempty"`
	Node      string  `json:"node"`
	Access    float64 `json:"access"`
	Consensus float64 `json:"consensus"`
}
