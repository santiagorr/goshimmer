// +build ignorepackage waspconn

import (
	"encoding/binary"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/packages/valuetransfers/packages/transaction"
	"github.com/pkg/errors"
	"io"
	"time"
)

func Uint16To2Bytes(val uint16) []byte {
	var tmp2 [2]byte
	binary.LittleEndian.PutUint16(tmp2[:], val)
	return tmp2[:]
}

func Uint32To4Bytes(val uint32) []byte {
	var tmp4 [4]byte
	binary.LittleEndian.PutUint32(tmp4[:], val)
	return tmp4[:]
}

func Uint32From4Bytes(b []byte) uint32 {
	if len(b) != 4 {
		panic("len(b) != 4")
	}
	return binary.LittleEndian.Uint32(b[:])
}

func Uint64To8Bytes(val uint64) []byte {
	var tmp8 [8]byte
	binary.LittleEndian.PutUint64(tmp8[:], val)
	return tmp8[:]
}

func WriteUint16(w io.Writer, val uint16) error {
	_, err := w.Write(Uint16To2Bytes(val))
	return err
}

func ReadUint16(r io.Reader, pval *uint16) error {
	var tmp2 [2]byte
	_, err := r.Read(tmp2[:])
	if err != nil {
		return err
	}
	*pval = binary.LittleEndian.Uint16(tmp2[:])
	return nil
}

func WriteUint32(w io.Writer, val uint32) error {
	_, err := w.Write(Uint32To4Bytes(val))
	return err
}

func ReadUint32(r io.Reader, pval *uint32) error {
	var tmp4 [4]byte
	_, err := r.Read(tmp4[:])
	if err != nil {
		return err
	}
	*pval = Uint32From4Bytes(tmp4[:])
	return nil
}

func WriteUint64(w io.Writer, val uint64) error {
	_, err := w.Write(Uint64To8Bytes(val))
	return err
}

func ReadByte(r io.Reader, pb *byte) error {
	var tmp [1]byte
	_, err := r.Read(tmp[:])
	if err != nil {
		return err
	}
	*pb = tmp[0]
	return nil
}

func WriteByte(w io.Writer, val byte) error {
	_, err := w.Write([]byte{val})
	return err
}

func ReadUint64(r io.Reader, pval *uint64) error {
	var tmp8 [8]byte
	_, err := r.Read(tmp8[:])
	if err != nil {
		return err
	}
	*pval = binary.LittleEndian.Uint64(tmp8[:])
	return nil
}

func WriteBoolByte(w io.Writer, cond bool) error {
	var b [1]byte
	if cond {
		b[0] = 0xFF
	}
	_, err := w.Write(b[:])
	return err
}

func ReadBoolByte(r io.Reader, cond *bool) error {
	var b [1]byte
	_, err := r.Read(b[:])
	if err != nil {
		return err
	}
	*cond = b[0] == 0xFF
	if !*cond && b[0] != 0x00 {
		return errors.New("ReadBoolByte: unexpected value")
	}
	return nil
}

func WriteTime(w io.Writer, ts time.Time) error {
	return WriteUint64(w, uint64(ts.UnixNano()))
}

func ReadTime(r io.Reader, ts *time.Time) error {
	var nano uint64
	err := ReadUint64(r, &nano)
	if err != nil {
		return err
	}
	*ts = time.Unix(0, int64(nano))
	return nil
}

const MaxUint16 = int(^uint16(0))

func WriteBytes16(w io.Writer, data []byte) error {
	if len(data) > MaxUint16 {
		panic("WriteBytes16: too long Data")
	}
	err := WriteUint16(w, uint16(len(data)))
	if err != nil {
		return err
	}
	if len(data) != 0 {
		_, err = w.Write(data)
	}
	return err
}

func ReadBytes16(r io.Reader) ([]byte, error) {
	var length uint16
	err := ReadUint16(r, &length)
	if err != nil {
		return nil, err
	}
	if length != 0 {
		ret := make([]byte, length)
		_, err = r.Read(ret)
		if err != nil {
			return nil, err
		}
		return ret, nil
	}
	return nil, nil
}

func WriteString(w io.Writer, s string) error {
	return WriteBytes16(w, []byte(s))
}

func ReadString(r io.Reader) (string, error) {
	ret, err := ReadBytes16(r)
	if err != nil {
		return "", err
	}
	return string(ret), nil
}

func WriteBytes32(w io.Writer, data []byte) error {
	err := WriteUint32(w, uint32(len(data)))
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func ReadBytes32(r io.Reader) ([]byte, error) {
	var length uint32
	err := ReadUint32(r, &length)
	if err != nil {
		return nil, err
	}
	ret := make([]byte, length)
	_, err = r.Read(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func ReadTransactionId(r io.Reader, txid *transaction.ID) error {
	n, err := r.Read(txid[:])
	if err != nil {
		return err
	}
	if n != transaction.IDLength {
		return errors.New("error while reading txid")
	}
	return nil
}

func ReadColor(r io.Reader, color *balance.Color) error {
	n, err := r.Read(color[:])
	if err != nil {
		return err
	}
	if n != balance.ColorLength {
		return errors.New("error while reading color")
	}
	return nil
}

func ReadAddress(r io.Reader, addr *address.Address) error {
	n, err := r.Read(addr[:])
	if err != nil {
		return err
	}
	if n != address.Length {
		return errors.New("error while reading address")
	}
	return nil
}

var niltxid transaction.ID

func ValidateBalances(outs map[transaction.ID][]*balance.Balance) error {
	for txid, bals := range outs {
		if txid == (transaction.ID)(balance.ColorNew) || txid == niltxid {
			return errors.New("ValidateBalances: wrong txid")
		}
		for _, b := range bals {
			if b.Color == balance.ColorNew {
				return errors.New("ValidateBalances: can't be ColorNew")
			}
			if b.Value <= 0 {
				return errors.New("ValidateBalances: can't be non-positive value")
			}
		}
	}
	return nil
}
