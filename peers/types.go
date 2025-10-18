package peers

import "net"

type PeerConnection struct {
	Conn net.Conn
	IP   string
	Port int
}
