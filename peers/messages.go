package peers

import (
	"encoding/binary"
	"io"
)

func (pc *PeerConnection) ReadMessage() (*Message, error) {
	lengthBytes := make([]byte, 4)
	_, err := io.ReadFull(pc.Conn, lengthBytes)
	if err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBytes)

	if length == 0 {
		return &Message{Type: 255}, nil
	}

	typeBytes := make([]byte, 1)
	_, err = io.ReadFull(pc.Conn, typeBytes)
	if err != nil {
		return nil, err
	}

	msgType := MessageType(typeBytes[0])

	payload := make([]byte, length-1)
	_, err = io.ReadFull(pc.Conn, payload)
	if err != nil {
		return nil, err
	}

	return &Message{
		Type:    msgType,
		Payload: payload,
	}, nil
}

func (pc *PeerConnection) SendMessage(msg *Message) error {
	payload := append([]byte{byte(msg.Type)}, msg.Payload...)
	length := uint32(len(payload))

	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, length)

	data := append(lengthBytes, payload...)
	_, err := pc.Conn.Write(data)
	return err
}

func (pc *PeerConnection) RequestPiece(pieceIndex int, offset int, length int) error {
	payload := make([]byte, 12)

	binary.BigEndian.PutUint32(payload[0:4], uint32(pieceIndex))

	binary.BigEndian.PutUint32(payload[4:8], uint32(offset))

	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	msg := &Message{
		Type:    Request,
		Payload: payload,
	}

	return pc.SendMessage(msg)
}
