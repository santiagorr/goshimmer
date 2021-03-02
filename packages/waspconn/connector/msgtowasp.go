package connector

// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

import (
	"io"

	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/transaction"
	"github.com/iotaledger/goshimmer/packages/waspconn"
	"github.com/iotaledger/goshimmer/packages/tangle"
)

func (wconn *WaspConnector) sendMsgToWasp(msg interface{ Write(io.Writer) error }) error {
	data, err := waspconn.EncodeMsg(msg)
	if err != nil {
		return err
	}
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
		dataToSend, err := waspconn.EncodeMsg(&waspconn.WaspMsgChunk{
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

func (wconn *WaspConnector) sendConfirmedTransactionToWasp(vtx *transaction.Transaction) error {
	return wconn.sendMsgToWasp(&waspconn.WaspFromNodeConfirmedTransactionMsg{
		Tx: vtx,
	})
}

func (wconn *WaspConnector) sendAddressUpdateToWasp(addr *address.Address, balances map[transaction.ID][]*balance.Balance, tx *transaction.Transaction) error {
	return wconn.sendMsgToWasp(&waspconn.WaspFromNodeAddressUpdateMsg{
		Address:  *addr,
		Balances: balances,
		Tx:       tx,
	})
}

func (wconn *WaspConnector) sendAddressOutputsToWasp(address *address.Address, balances map[transaction.ID][]*balance.Balance) error {
	return wconn.sendMsgToWasp(&waspconn.WaspFromNodeAddressOutputsMsg{
		Address:  *address,
		Balances: balances,
	})
}

func (wconn *WaspConnector) sendTxInclusionLevelToWasp(inclLevel byte, txid *transaction.ID, addrs []address.Address) error {
	return wconn.sendMsgToWasp(&waspconn.WaspFromNodeTransactionInclusionLevelMsg{
		Level:               inclLevel,
		TxId:                *txid,
		SubscribedAddresses: addrs,
	})
}

// query outputs database and collects transactions containing unprocessed requests
func (wconn *WaspConnector) pushBacklogToWasp(addr *address.Address, scColor *balance.Color) {
	outs, err := wconn.vtangle.GetConfirmedAddressOutputs(*addr)
	if err != nil {
		wconn.log.Errorf("pushBacklogToWasp: %v", err)
		return
	}
	if len(outs) == 0 {
		return
	}
	balancesByTx := waspconn.OutputsToBalances(outs)
	balancesByColor, _ := waspconn.OutputBalancesByColor(outs)

	wconn.log.Debugf("pushBacklogToWasp: balancesByTx of addr %s by transaction:\n%s",
		addr.String(), waspconn.OutputsByTransactionToString(balancesByTx))

	wconn.log.Debugf("pushBacklogToWasp: balancesByTx of addr %s by color:\n%s",
		addr.String(), waspconn.BalancesByColorToString(balancesByColor))

	allColorsAsTxid := make([]transaction.ID, 0, len(balancesByColor))
	for col, b := range balancesByColor {
		if col == balance.ColorIOTA {
			continue
		}
		if col == balance.ColorNew {
			wconn.log.Warnf("pushBacklogToWasp: unexpected ColorNew encountered in the balancesByTx of address %s", addr.String())
			continue
		}
		if col == *scColor && b == 1 {
			// color of the scColor belongs to backlog only if more than 1 token
			continue
		}
		allColorsAsTxid = append(allColorsAsTxid, (transaction.ID)(col))
	}
	wconn.log.Debugf("pushBacklogToWasp: color candidates for request transactions: %+v\n", allColorsAsTxid)

	// for each color we try to load corresponding origin transaction.
	// if the transaction exist and it is among the balancesByTx of the address,
	// then send balancesByTx with the transaction as address update
	sentTxs := make([]transaction.ID, 0)
	for _, txid := range allColorsAsTxid {
		tx := wconn.vtangle.GetConfirmedTransaction(&txid)
		if tx == nil {
			wconn.log.Warnf("pushBacklogToWasp: can't find the origin tx for the color %s. It may be snapshotted", txid.String())
			continue
		}

		wconn.log.Debugf("pushBacklogToWasp: sending update with txid: %s\n", tx.ID().String())

		if err := wconn.sendAddressUpdateToWasp(addr, balancesByTx, tx); err != nil {
			wconn.log.Errorf("pushBacklogToWasp:sendAddressUpdateToWasp: %v", err)
		} else {
			sentTxs = append(sentTxs, txid)
		}
	}
	wconn.log.Infof("pushed backlog to Wasp for addr %s. sent transactions: %+v", addr.String(), sentTxs)
}
