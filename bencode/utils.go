package bencode

import (
	"fmt"
	"strconv"
)

func getNumberTilStopper(data []byte, pos int, stopper byte) (int64, int, error) {
	start := pos

	if data[pos] == '-' {
		pos++
	}
	for pos < len(data) && data[pos] != stopper {
		if data[pos] < '0' || data[pos] > '9' {
			return -1, pos, fmt.Errorf("invalid digit: %d", data[pos])
		}
		pos++
	}
	if pos >= len(data) || data[pos] != stopper {
		return -1, pos, fmt.Errorf("value not terminated with '%c'", stopper)
	}

	num, err := strconv.ParseInt(string(data[start:pos]), 10, 64)
	if err != nil {
		return -1, pos, fmt.Errorf("failed to parse number: %w", err)
	}
	return num, pos, nil
}
