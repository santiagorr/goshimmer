package connector

// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

import (
	"github.com/iotaledger/goshimmer/packages/tangle"
	"github.com/iotaledger/goshimmer/packages/waspconn"
)

// process messages received from the Wasp
func (wconn *WaspConnector) processMsgDataFromWasp(data []byte) {
	var msg interface{}
	var err error
	if msg, err = waspconn.DecodeMsg(data, false); err != nil {
		wconn.log.Errorf("DecodeMsg: %v", err)
		return
	}
	switch msg := msg.(type) {
	case *waspconn.WaspMsgChunk:
		finalMsg, err := wconn.messageChopper.IncomingChunk(msg.Data, tangle.MaxMessageSize, waspconn.ChunkMessageHeaderSize)
		if err != nil {
			wconn.log.Errorf("DecodeMsg: %v", err)
			return
		}
		if finalMsg != nil {
			wconn.processMsgDataFromWasp(finalMsg)
		}

	case *waspconn.WaspPingMsg:
		wconn.log.Debugf("PING %d received", msg.Id)
		if err := wconn.sendMsgToWasp(msg); err != nil {
			wconn.log.Errorf("responding to ping: %v", err)
		}

	case *waspconn.WaspToNodeTransactionMsg:
		wconn.postTransaction(msg.Tx, msg.SCAddress, msg.Leader)

	case *waspconn.WaspToNodeSubscribeMsg:
		for _, addrCol := range msg.AddressesWithColors {
			wconn.subscribe(addrCol.Address, addrCol.Color)
		}
		go func() {
			for _, addrCol := range msg.AddressesWithColors {
				wconn.pushBacklogToWasp(addrCol.Address, addrCol.Color)
			}
		}()

	case *waspconn.WaspToNodeGetConfirmedTransactionMsg:
		wconn.getConfirmedTransaction(msg.TxId)

	case *waspconn.WaspToNodeGetBranchInclusionStateMsg:
		wconn.getBranchInclusionState(msg.TxId, msg.SCAddress)

	case *waspconn.WaspToNodeGetOutputsMsg:
		wconn.getAddressBalance(msg.Address)

	case *waspconn.WaspToNodeSetIdMsg:
		wconn.SetId(msg.Waspid)

	default:
		panic("wrong msg type")
	}
}
