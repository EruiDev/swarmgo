package peers

import "net"

type PeerConnection struct {
	Conn net.Conn
	IP   string
	Port int
}

type MessageType uint8

const (
	Choke         MessageType = 0
	Unchoke       MessageType = 1
	Interested    MessageType = 2
	NotInterested MessageType = 3
	Have          MessageType = 4
	Bitfield      MessageType = 5
	Request       MessageType = 6
	Piece         MessageType = 7
	Cancel        MessageType = 8
)

type Message struct {
	Type    MessageType
	Payload []byte
}
