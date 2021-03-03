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
	switch msgt := msg.(type) {
	case *waspconn.WaspMsgChunk:
		finalMsg, err := wconn.messageChopper.IncomingChunk(msgt.Data, tangle.MaxMessageSize, waspconn.ChunkMessageHeaderSize)
		if err != nil {
			wconn.log.Errorf("DecodeMsg: %v", err)
			return
		}
		if finalMsg != nil {
			wconn.processMsgDataFromWasp(finalMsg)
		}

	case *waspconn.WaspPingMsg:
		wconn.log.Debugf("PING %d received", msgt.Id)
		if err := wconn.sendMsgToWasp(msgt); err != nil {
			wconn.log.Errorf("responding to ping: %v", err)
		}

	case *waspconn.WaspToNodeTransactionMsg:
		wconn.postTransaction(msgt.Tx, msgt.SCAddress, msgt.Leader)

	case *waspconn.WaspToNodeSubscribeMsg:
		for _, addrCol := range msgt.AddressesWithColors {
			wconn.subscribe(addrCol.Address, addrCol.Color)
		}
		go func() {
			for _, addrCol := range msgt.AddressesWithColors {
				wconn.pushBacklogToWasp(addrCol.Address, &addrCol.Color)
			}
		}()

	case *waspconn.WaspToNodeGetConfirmedTransactionMsg:
		wconn.getConfirmedTransaction(msgt.TxId)

	case *waspconn.WaspToNodeGetBranchInclusionStateMsg:
		wconn.getBranchInclusionState(msgt.TxId, msgt.SCAddress)

	case *waspconn.WaspToNodeGetOutputsMsg:
		wconn.getAddressBalance(msgt.Address)

	case *waspconn.WaspToNodeSetIdMsg:
		wconn.SetId(msgt.Waspid)

	default:
		panic("wrong msg type")
	}
}
