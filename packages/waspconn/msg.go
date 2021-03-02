// +build ignorepackage waspconn

import (
	"bytes"
	"fmt"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/transaction"
	"io"
	"sort"
)

const (
	waspPing = iota
	waspMsgChunk

	// wasp -> node
	waspToNodeTransaction
	waspToNodeSubscribe
	waspToNodeGetConfirmedTransaction
	waspToNodeGetTxInclusionLevel
	waspToNodeGetOutputs
	waspToNodeSetId

	// node -> wasp
	waspFromNodeConfirmedTransaction
	waspFromNodeAddressUpdate
	waspFromNodeAddressOutputs
	waspFromNodeTransactionInclusionState
)

const ChunkMessageHeaderSize = 3

// special messages for big Data packets chopped into pieces
type WaspMsgChunk struct {
	Data []byte
}

type WaspPingMsg struct {
	Id        uint32
	Timestamp int64
}

type WaspToNodeTransactionMsg struct {
	Tx        *transaction.Transaction // transaction posted
	SCAddress address.Address          // smart contract which posted
	Leader    uint16                   // leader index
}

type AddressColor struct {
	Address address.Address
	Color   balance.Color
}
type WaspToNodeSubscribeMsg struct {
	AddressesWithColors []AddressColor
}

type WaspToNodeGetConfirmedTransactionMsg struct {
	TxId transaction.ID
}

type WaspToNodeGetTxInclusionLevelMsg struct {
	TxId      transaction.ID
	SCAddress address.Address
}

type WaspToNodeGetOutputsMsg struct {
	Address address.Address
}

type WaspToNodeSetIdMsg struct {
	Waspid string
}

type WaspFromNodeConfirmedTransactionMsg struct {
	Tx *transaction.Transaction
}

type WaspFromNodeAddressUpdateMsg struct {
	Address  address.Address
	Balances map[transaction.ID][]*balance.Balance
	Tx       *transaction.Transaction
}

type WaspFromNodeAddressOutputsMsg struct {
	Address  address.Address
	Balances map[transaction.ID][]*balance.Balance
}

const (
	TransactionInclusionLevelUndef = iota
	TransactionInclusionLevelBooked
	TransactionInclusionLevelConfirmed
	TransactionInclusionLevelRejected
)

type WaspFromNodeTransactionInclusionLevelMsg struct {
	Level               byte
	TxId                transaction.ID
	SubscribedAddresses []address.Address // addresses which transaction might be interesting to
}

func typeToCode(msg interface{ Write(writer io.Writer) error }) byte {
	switch msg.(type) {
	case *WaspPingMsg:
		return waspPing

	case *WaspMsgChunk:
		return waspMsgChunk

	case *WaspToNodeTransactionMsg:
		return waspToNodeTransaction

	case *WaspToNodeSubscribeMsg:
		return waspToNodeSubscribe

	case *WaspToNodeGetConfirmedTransactionMsg:
		return waspToNodeGetConfirmedTransaction

	case *WaspToNodeGetTxInclusionLevelMsg:
		return waspToNodeGetTxInclusionLevel

	case *WaspToNodeGetOutputsMsg:
		return waspToNodeGetOutputs

	case *WaspToNodeSetIdMsg:
		return waspToNodeSetId

	case *WaspFromNodeConfirmedTransactionMsg:
		return waspFromNodeConfirmedTransaction

	case *WaspFromNodeAddressUpdateMsg:
		return waspFromNodeAddressUpdate

	case *WaspFromNodeAddressOutputsMsg:
		return waspFromNodeAddressOutputs

	case *WaspFromNodeTransactionInclusionLevelMsg:
		return waspFromNodeTransactionInclusionState
	}

	panic("wrong type")
}

func EncodeMsg(msg interface{ Write(writer io.Writer) error }) ([]byte, error) {
	msgCode := typeToCode(msg)
	var buf bytes.Buffer

	if err := buf.WriteByte(msgCode); err != nil {
		return nil, err
	}
	if err := msg.Write(&buf); err != nil {
		return nil, err
	}
	ret := buf.Bytes()
	return ret, nil
}

func DecodeMsg(data []byte, waspSide bool) (interface{}, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("wrong message")
	}
	var ret interface{ Read(io.Reader) error }

	switch data[0] {
	case waspPing:
		ret = &WaspPingMsg{}

	case waspMsgChunk:
		ret = &WaspMsgChunk{}

	case waspToNodeTransaction:
		if waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspToNodeTransactionMsg{}

	case waspToNodeSubscribe:
		if waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspToNodeSubscribeMsg{}

	case waspToNodeGetConfirmedTransaction:
		if waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspToNodeGetConfirmedTransactionMsg{}

	case waspToNodeGetTxInclusionLevel:
		if waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspToNodeGetTxInclusionLevelMsg{}

	case waspToNodeGetOutputs:
		if waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspToNodeGetOutputsMsg{}

	case waspToNodeSetId:
		if waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspToNodeSetIdMsg{}

	case waspFromNodeConfirmedTransaction:
		if !waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspFromNodeConfirmedTransactionMsg{}

	case waspFromNodeAddressUpdate:
		if !waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspFromNodeAddressUpdateMsg{}

	case waspFromNodeAddressOutputs:
		if !waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspFromNodeAddressOutputsMsg{}

	case waspFromNodeTransactionInclusionState:
		if !waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspFromNodeTransactionInclusionLevelMsg{}

	default:
		return nil, fmt.Errorf("wrong message code")
	}
	if err := ret.Read(bytes.NewReader(data[1:])); err != nil {
		return nil, err
	}
	return ret, nil
}

func (msg *WaspPingMsg) Write(w io.Writer) error {
	if err := WriteUint32(w, msg.Id); err != nil {
		return err
	}
	if err := WriteUint64(w, uint64(msg.Timestamp)); err != nil {
		return err
	}
	return nil
}

func (msg *WaspPingMsg) Read(r io.Reader) error {
	if err := ReadUint32(r, &msg.Id); err != nil {
		return err
	}
	var ts uint64
	if err := ReadUint64(r, &ts); err != nil {
		return err
	}
	msg.Timestamp = int64(ts)
	return nil
}

func (msg *WaspToNodeTransactionMsg) Write(w io.Writer) error {
	if err := WriteBytes32(w, msg.Tx.Bytes()); err != nil {
		return err
	}
	if _, err := w.Write(msg.SCAddress[:]); err != nil {
		return err
	}
	if err := WriteUint16(w, msg.Leader); err != nil {
		return err
	}
	return nil
}

func (msg *WaspToNodeTransactionMsg) Read(r io.Reader) error {
	var err error
	data, err := ReadBytes32(r)
	if err != nil {
		return err
	}
	msg.Tx, _, err = transaction.FromBytes(data)
	if err != nil {
		return err
	}
	if err := ReadAddress(r, &msg.SCAddress); err != nil {
		return err
	}
	if err := ReadUint16(r, &msg.Leader); err != nil {
		return err
	}
	return nil
}

func (msg *WaspToNodeSubscribeMsg) Write(w io.Writer) error {
	if err := WriteUint16(w, uint16(len(msg.AddressesWithColors))); err != nil {
		return err
	}
	for _, addrCol := range msg.AddressesWithColors {
		if _, err := w.Write(addrCol.Address[:]); err != nil {
			return err
		}
		if _, err := w.Write(addrCol.Color[:]); err != nil {
			return err
		}
	}
	return nil
}

func (msg *WaspToNodeSubscribeMsg) Read(r io.Reader) error {
	var size uint16
	if err := ReadUint16(r, &size); err != nil {
		return err
	}
	msg.AddressesWithColors = make([]AddressColor, size)
	for i := range msg.AddressesWithColors {
		if err := ReadAddress(r, &msg.AddressesWithColors[i].Address); err != nil {
			return err
		}
		if err := ReadColor(r, &msg.AddressesWithColors[i].Color); err != nil {
			return err
		}
	}
	return nil
}

func (msg *WaspToNodeGetConfirmedTransactionMsg) Write(w io.Writer) error {
	_, err := w.Write(msg.TxId[:])
	return err
}

func (msg *WaspToNodeGetConfirmedTransactionMsg) Read(r io.Reader) error {
	return ReadTransactionId(r, &msg.TxId)
}

func (msg *WaspToNodeGetTxInclusionLevelMsg) Write(w io.Writer) error {
	if _, err := w.Write(msg.TxId[:]); err != nil {
		return err
	}
	if _, err := w.Write(msg.SCAddress[:]); err != nil {
		return err
	}
	return nil
}

func (msg *WaspToNodeGetTxInclusionLevelMsg) Read(r io.Reader) error {
	if err := ReadTransactionId(r, &msg.TxId); err != nil {
		return err
	}
	if err := ReadAddress(r, &msg.SCAddress); err != nil {
		return err
	}
	return nil
}

func (msg *WaspToNodeGetOutputsMsg) Write(w io.Writer) error {
	_, err := w.Write(msg.Address[:])
	return err
}

func (msg *WaspToNodeGetOutputsMsg) Read(r io.Reader) error {
	return ReadAddress(r, &msg.Address)
}

func (msg *WaspToNodeSetIdMsg) Write(w io.Writer) error {
	return WriteString(w, msg.Waspid)
}

func (msg *WaspToNodeSetIdMsg) Read(r io.Reader) error {
	var err error
	msg.Waspid, err = ReadString(r)
	return err
}

func (msg *WaspFromNodeConfirmedTransactionMsg) Write(w io.Writer) error {
	if err := WriteBytes32(w, msg.Tx.Bytes()); err != nil {
		return err
	}
	return nil
}

func (msg *WaspFromNodeConfirmedTransactionMsg) Read(r io.Reader) error {
	data, err := ReadBytes32(r)
	if err != nil {
		return err
	}
	if msg.Tx, _, err = transaction.FromBytes(data); err != nil {
		return err
	}
	return nil
}

func (msg *WaspFromNodeAddressUpdateMsg) Write(w io.Writer) error {
	_, err := w.Write(msg.Address[:])
	if err != nil {
		return err
	}
	if err := WriteBalances(w, msg.Balances); err != nil {
		return err
	}
	return WriteBytes32(w, msg.Tx.Bytes())
}

func (msg *WaspFromNodeAddressUpdateMsg) Read(r io.Reader) error {
	var err error
	if err = ReadAddress(r, &msg.Address); err != nil {
		return err
	}
	if msg.Balances, err = ReadBalances(r); err != nil {
		return err
	}
	data, err := ReadBytes32(r)
	if err != nil {
		return err
	}
	msg.Tx, _, err = transaction.FromBytes(data)
	return err
}

func (msg *WaspFromNodeAddressOutputsMsg) Write(w io.Writer) error {
	_, err := w.Write(msg.Address[:])
	if err != nil {
		return err
	}
	return WriteBalances(w, msg.Balances)
}

func (msg *WaspMsgChunk) Read(r io.Reader) error {
	var err error
	msg.Data, err = ReadBytes16(r)
	return err
}

func (msg *WaspMsgChunk) Write(w io.Writer) error {
	return WriteBytes16(w, msg.Data)
}

func (msg *WaspFromNodeAddressOutputsMsg) Read(r io.Reader) error {
	var err error
	if err = ReadAddress(r, &msg.Address); err != nil {
		return err
	}
	msg.Balances, err = ReadBalances(r)
	return err
}

func (msg *WaspFromNodeTransactionInclusionLevelMsg) Write(w io.Writer) error {
	if err := WriteByte(w, msg.Level); err != nil {
		return err
	}
	if _, err := w.Write(msg.TxId[:]); err != nil {
		return err
	}
	numAddrs := uint16(len(msg.SubscribedAddresses))
	if err := WriteUint16(w, numAddrs); err != nil {
		return err
	}
	for i := range msg.SubscribedAddresses {
		if _, err := w.Write(msg.SubscribedAddresses[i][:]); err != nil {
			return err
		}
	}
	return nil
}

func (msg *WaspFromNodeTransactionInclusionLevelMsg) Read(r io.Reader) error {
	if err := ReadByte(r, &msg.Level); err != nil {
		return err
	}
	if err := ReadTransactionId(r, &msg.TxId); err != nil {
		return err
	}
	var numAddrs uint16
	if err := ReadUint16(r, &numAddrs); err != nil {
		return err
	}
	msg.SubscribedAddresses = make([]address.Address, numAddrs)
	for i := range msg.SubscribedAddresses {
		if err := ReadAddress(r, &msg.SubscribedAddresses[i]); err != nil {
			return err
		}
	}
	return nil
}

func WriteBalances(w io.Writer, balances map[transaction.ID][]*balance.Balance) error {
	if err := ValidateBalances(balances); err != nil {
		return err
	}
	if err := WriteUint16(w, uint16(len(balances))); err != nil {
		return err
	}
	for txid, bals := range balances {
		if _, err := w.Write(txid[:]); err != nil {
			return err
		}
		if err := WriteUint16(w, uint16(len(bals))); err != nil {
			return err
		}
		for _, b := range bals {
			if _, err := w.Write(b.Color[:]); err != nil {
				return err
			}
			if err := WriteUint64(w, uint64(b.Value)); err != nil {
				return err
			}
		}
	}
	return nil
}

func ReadBalances(r io.Reader) (map[transaction.ID][]*balance.Balance, error) {
	var size uint16
	if err := ReadUint16(r, &size); err != nil {
		return nil, err
	}
	ret := make(map[transaction.ID][]*balance.Balance, size)
	for i := uint16(0); i < size; i++ {
		var txid transaction.ID
		if err := ReadTransactionId(r, &txid); err != nil {
			return nil, err
		}
		var numBals uint16
		if err := ReadUint16(r, &numBals); err != nil {
			return nil, err
		}
		lst := make([]*balance.Balance, numBals)
		for i := range lst {
			var color balance.Color
			if err := ReadColor(r, &color); err != nil {
				return nil, err
			}
			var value uint64
			if err := ReadUint64(r, &value); err != nil {
				return nil, err
			}
			lst[i] = balance.New(color, int64(value))
		}
		ret[txid] = lst
	}
	if err := ValidateBalances(ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func OutputsToBalances(outs map[transaction.OutputID][]*balance.Balance) map[transaction.ID][]*balance.Balance {
	ret := make(map[transaction.ID][]*balance.Balance)
	var niltxid transaction.ID

	for outp, bals := range outs {
		if outp.TransactionID() == niltxid {
			panic("outp.TransactionID() == niltxid")
		}
		ret[outp.TransactionID()] = bals
	}
	return ret
}

func BalancesToOutputs(addr address.Address, bals map[transaction.ID][]*balance.Balance) map[transaction.OutputID][]*balance.Balance {
	ret := make(map[transaction.OutputID][]*balance.Balance)
	var niltxid transaction.ID

	for txid, bal := range bals {
		if txid == niltxid {
			panic("txid == niltxid")
		}
		ret[transaction.NewOutputID(addr, txid)] = bal
	}
	return ret
}

func OutputBalancesByColor(outs map[transaction.OutputID][]*balance.Balance) (map[balance.Color]int64, int64) {
	ret := make(map[balance.Color]int64)
	var total int64
	for _, bals := range outs {
		for _, b := range bals {
			if s, ok := ret[b.Color]; !ok {
				ret[b.Color] = b.Value
			} else {
				ret[b.Color] = s + b.Value
			}
			total += b.Value
		}
	}
	return ret, total
}

func OutputsByTransactionToString(outs map[transaction.ID][]*balance.Balance) string {
	ret := ""
	for txid, bals := range outs {
		ret += fmt.Sprintf("     %s:\n", txid.String())
		for _, b := range bals {
			ret += fmt.Sprintf("            %s: %d\n", b.Color.String(), b.Value)
		}
	}
	return ret
}

func BalancesByColorToString(bals map[balance.Color]int64) string {
	ret := ""
	for col, b := range bals {
		ret += fmt.Sprintf("      %s: %d\n", col.String(), b)
	}
	return ret
}

// utility function for testing
var inclusionLevels = map[byte]string{
	TransactionInclusionLevelUndef:     "undef",
	TransactionInclusionLevelBooked:    "booked",
	TransactionInclusionLevelConfirmed: "confirmed",
	TransactionInclusionLevelRejected:  "rejected",
}

// InclusionLevelText return text representation of the code
func InclusionLevelText(level byte) string {
	if ret, ok := inclusionLevels[level]; ok {
		return ret
	}
	return "wrong code"
}

func BalancesToString(outs map[transaction.ID][]*balance.Balance) string {
	if outs == nil {
		return "empty balances"
	}

	txids := make([]transaction.ID, 0, len(outs))
	for txid := range outs {
		txids = append(txids, txid)
	}
	sort.Slice(txids, func(i, j int) bool {
		return bytes.Compare(txids[i][:], txids[j][:]) < 0
	})

	ret := ""
	for _, txid := range txids {
		bals := outs[txid]
		ret += txid.String() + ":\n"
		for _, bal := range bals {
			ret += fmt.Sprintf("         %s: %d\n", bal.Color.String(), bal.Value)
		}
	}
	return ret
}
