package models

// git PKT-Line api
// PktLineType message type of pkt-line
type PktLineType int64

const (
	// Unknow type
	PktLineTypeUnknow PktLineType = 0
	// flush-pkt "0000"
	PktLineTypeFlush PktLineType = iota
	// data line
	PktLineTypeData
)

// GitPktLine pkt-line api
type GitPktLine struct {
	Type   PktLineType
	Length uint64
	Data   []byte
}
