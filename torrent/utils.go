package torrent

import (
	"crypto/rand"
	"fmt"
)

func GeneratePeerID() ([20]byte, error) {
	var peerID [20]byte
	copy(peerID[:], "-GO0001-")

	_, err := rand.Read(peerID[8:])
	if err != nil {
		return [20]byte{}, fmt.Errorf("error reading")
	}
	return peerID, nil
}
