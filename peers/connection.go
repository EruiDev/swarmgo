package peers

import (
	"bytes"
	"fmt"
	"net"
	"time"
)

func Connect(ip string, port int) (*PeerConnection, error) {
	addr := fmt.Sprintf("%s:%d", ip, port)

	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, err
	}

	return &PeerConnection{
		Conn: conn,
		IP:   ip,
		Port: port,
	}, nil
}

func (pc *PeerConnection) Handshake(infoHash [20]byte, peerID [20]byte) error {
	handshake := make([]byte, 68)
	handshake[0] = 19
	copy(handshake[1:20], []byte("BitTorrent protocol"))
	copy(handshake[28:48], infoHash[:])
	copy(handshake[48:68], peerID[:])

	_, err := pc.Conn.Write(handshake)
	if err != nil {
		return err
	}

	resp := make([]byte, 68)
	pc.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = pc.Conn.Read(resp)
	if err != nil {
		return err
	}

	if !bytes.Equal(resp[28:48], infoHash[:]) {
		return fmt.Errorf("info hash mismatch")
	}

	return nil
}
