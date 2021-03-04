package connector

// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

import (
	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"github.com/iotaledger/goshimmer/packages/tangle"
	"github.com/iotaledger/goshimmer/packages/waspconn"
)

func (wconn *WaspConnector) sendMsgToWasp(msg waspconn.Message) error {
	data := waspconn.EncodeMsg(msg)
	choppedData, chopped, err := wconn.messageChopper.ChopData(data, tangle.MaxMessageSize, waspconn.ChunkMessageHeaderSize)
	if err != nil {
		return err
	}
	if !chopped {
		_, err = wconn.bconn.Write(data)
		return err
	}

	wconn.log.Debugf("+++++++++++++ %d bytes long message was split into %d chunks", len(data), len(choppedData))

	// sending piece by piece wrapped in WaspMsgChunk
	for _, piece := range choppedData {
		dataToSend := waspconn.EncodeMsg(&waspconn.WaspMsgChunk{
			Data: piece,
		})
		if err != nil {
			return err
		}
		if len(dataToSend) > tangle.MaxMessageSize {
			wconn.log.Panicf("sendMsgToWasp: internal inconsistency 3 size too big: %d", len(dataToSend))
		}
		_, err = wconn.bconn.Write(dataToSend)
		if err != nil {
			return err
		}
	}
	return nil
}

func (wconn *WaspConnector) sendConfirmedTransactionToWasp(vtx *ledgerstate.Transaction) error {
	return wconn.sendMsgToWasp(&waspconn.WaspFromNodeConfirmedTransactionMsg{
		Tx: vtx,
	})
}

func (wconn *WaspConnector) sendAddressUpdateToWasp(addr ledgerstate.Address, balances map[ledgerstate.TransactionID]*ledgerstate.ColoredBalances, tx *ledgerstate.Transaction) error {
	return wconn.sendMsgToWasp(&waspconn.WaspFromNodeAddressUpdateMsg{
		Address:  addr,
		Balances: balances,
		Tx:       tx,
	})
}

func (wconn *WaspConnector) sendAddressOutputsToWasp(address ledgerstate.Address, balances map[ledgerstate.TransactionID]*ledgerstate.ColoredBalances) error {
	return wconn.sendMsgToWasp(&waspconn.WaspFromNodeAddressOutputsMsg{
		Address:  address,
		Balances: balances,
	})
}

func (wconn *WaspConnector) sendBranchInclusionStateToWasp(state ledgerstate.InclusionState, txid ledgerstate.TransactionID, addrs []ledgerstate.Address) error {
	return wconn.sendMsgToWasp(&waspconn.WaspFromNodeBranchInclusionStateMsg{
		State:               state,
		TxId:                txid,
		SubscribedAddresses: addrs,
	})
}

// query outputs database and collects transactions containing unprocessed requests
func (wconn *WaspConnector) pushBacklogToWasp(addr ledgerstate.Address, scColor ledgerstate.Color) {
	outs := wconn.vtangle.GetAddressOutputs(addr)
	if len(outs) == 0 {
		return
	}
	balancesByTx := waspconn.OutputsToBalances(outs)
	balancesByColor, _ := waspconn.OutputBalancesByColor(outs)

	wconn.log.Debugf("pushBacklogToWasp: balancesByTx of addr %s by transaction:\n%s",
		addr.String(), waspconn.OutputsByTransactionToString(balancesByTx))

	wconn.log.Debugf("pushBacklogToWasp: balancesByTx of addr %s by color:\n%s",
		addr.String(), waspconn.BalancesByColorToString(balancesByColor))

	allColorsAsTxid := make([]ledgerstate.TransactionID, 0, len(balancesByColor))
	for col, b := range balancesByColor {
		if col == ledgerstate.ColorIOTA {
			continue
		}
		if col == ledgerstate.ColorMint {
			wconn.log.Warnf("pushBacklogToWasp: unexpected ColorMint encountered in the balancesByTx of address %s", addr.String())
			continue
		}
		if col == scColor && b == 1 {
			// color of the scColor belongs to backlog only if more than 1 token
			continue
		}
		allColorsAsTxid = append(allColorsAsTxid, (ledgerstate.TransactionID)(col))
	}
	wconn.log.Debugf("pushBacklogToWasp: color candidates for request transactions: %+v\n", allColorsAsTxid)

	// for each color we try to load corresponding origin transaction.
	// if the transaction exist and it is among the balancesByTx of the address,
	// then send balancesByTx with the transaction as address update
	sentTxs := make([]ledgerstate.TransactionID, 0)
	for _, txid := range allColorsAsTxid {
		found := wconn.vtangle.GetTransaction(txid, func(tx *ledgerstate.Transaction) {
			state, _ := wconn.vtangle.GetBranchInclusionState(txid)
			if state != ledgerstate.Confirmed {
				wconn.log.Warnf("pushBacklogToWasp: not confirmed %s", txid.String())
				return
			}

			wconn.log.Debugf("pushBacklogToWasp: sending update with txid: %s\n", tx.ID().String())

			if err := wconn.sendAddressUpdateToWasp(addr, balancesByTx, tx); err != nil {
				wconn.log.Errorf("pushBacklogToWasp:sendAddressUpdateToWasp: %v", err)
			} else {
				sentTxs = append(sentTxs, txid)
			}
		})
		if !found {
			wconn.log.Warnf("pushBacklogToWasp: can't find the origin tx for the color %s. It may be snapshotted", txid.String())
		}
	}
	wconn.log.Infof("pushed backlog to Wasp for addr %s. sent transactions: %+v", addr.String(), sentTxs)
}
