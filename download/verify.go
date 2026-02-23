package download

import "crypto/sha1"

func verifyPiece(data []byte, expectedHash []byte) bool {
	hash := sha1.Sum(data)
	return string(hash[:]) == string(expectedHash)
}
