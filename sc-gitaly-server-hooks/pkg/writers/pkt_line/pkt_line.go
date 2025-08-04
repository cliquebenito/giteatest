package pkt_line

import (
	"fmt"
	"io"
)

type Writer struct{}

func NewPktLineWriter() Writer {
	return Writer{}
}

func (writer Writer) WriteFlushPktLine(out io.Writer) error {
	l, err := out.Write([]byte("0000"))
	if err != nil || l != 4 {
		return fmt.Errorf("protocol: write error.\nPkt-Line response failed: %v", err)
	}
	return nil
}

func (writer Writer) WriteDataPktLine(out io.Writer, data []byte) error {
	hexchar := []byte("0123456789abcdef")
	hex := func(n uint64) byte {
		return hexchar[(n)&15]
	}

	length := uint64(len(data) + 4)
	tmp := make([]byte, 4)
	tmp[0] = hex(length >> 12)
	tmp[1] = hex(length >> 8)
	tmp[2] = hex(length >> 4)
	tmp[3] = hex(length)

	lr, err := out.Write(tmp)
	if err != nil || lr != 4 {
		return fmt.Errorf("protocol: write error.\nPkt-Line response failed: %v", err)
	}

	lr, err = out.Write(data)
	if err != nil || int(length-4) != lr {
		return fmt.Errorf("protocol: write error.\nPkt-Line response failed: %v", err)
	}

	return nil
}
