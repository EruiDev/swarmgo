package bencode

import (
	"crypto/sha1"
	"fmt"
)

func GetInfoHash(val Value) ([20]byte, error) {
	infoVal, ok := val.(Dict)["info"]
	if !ok {
		return [20]byte{}, fmt.Errorf("missing info dict")
	}

	infoBytes, err := Encode(infoVal)
	if err != nil {
		return [20]byte{}, err
	}

	hash := sha1.Sum(infoBytes)
	return hash, nil
}
