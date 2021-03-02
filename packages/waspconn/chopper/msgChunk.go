package chopper

// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

import (
	"bytes"
	"fmt"

	"github.com/iotaledger/goshimmer/packages/waspconn"
)

// special wrapper message for chunks of larger than buffer messages
type msgChunk struct {
	msgId       uint32
	chunkSeqNum byte
	numChunks   byte
	data        []byte
}

func (c *msgChunk) encode() []byte {
	var buf bytes.Buffer

	_ = waspconn.WriteUint32(&buf, c.msgId)
	_ = waspconn.WriteByte(&buf, c.numChunks)
	_ = waspconn.WriteByte(&buf, c.chunkSeqNum)
	_ = waspconn.WriteBytes16(&buf, c.data)
	return buf.Bytes()
}

func (c *msgChunk) decode(data []byte, maxChunkSizeWithoutHeader int) error {
	rdr := bytes.NewReader(data)
	if err := waspconn.ReadUint32(rdr, &c.msgId); err != nil {
		return err
	}
	if err := waspconn.ReadByte(rdr, &c.numChunks); err != nil {
		return err
	}
	if err := waspconn.ReadByte(rdr, &c.chunkSeqNum); err != nil {
		return err
	}
	if data, err := waspconn.ReadBytes16(rdr); err != nil {
		return err
	} else {
		c.data = data
	}
	if c.chunkSeqNum >= c.numChunks {
		return fmt.Errorf("wrong data chunk format")
	}
	if len(c.data) != maxChunkSizeWithoutHeader && c.chunkSeqNum != c.numChunks-1 {
		return fmt.Errorf("wrong data chunk length")
	}
	return nil
}
