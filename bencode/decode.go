package bencode

import (
	"fmt"
)

func decodeInteger(data []byte, pos int) (Value, int, error) {
	pos++
	num, pos, err := getNumberTilStopper(data, pos, 'e')
	if err != nil {
		return nil, pos, err
	}
	return Integer(num), pos + 1, nil
}

func decodeString(data []byte, pos int) (Value, int, error) {
	num, pos, err := getNumberTilStopper(data, pos, ':')

	if err != nil {
		return nil, pos, err
	}
	if num < 0 {
		return nil, pos, fmt.Errorf("string length can't be negative")
	}

	pos++
	end := int(num) + pos

	if end > len(data) {
		return nil, pos, fmt.Errorf("invalid length of string, out of bounds")
	}
	str := string(data[pos:end])
	return String(str), end, nil
}

func decodeList(data []byte, pos int) (Value, int, error) {
	pos++
	list := List{}

	for pos < len(data) && data[pos] != 'e' {
		val, newPos, err := decodeValue(data, pos)
		if err != nil {
			return nil, pos, err
		}
		list = append(list, val)
		pos = newPos
	}

	if pos >= len(data) || data[pos] != 'e' {
		return nil, pos, fmt.Errorf("list not terminated with 'e'")
	}

	pos++
	return list, pos, nil
}

func decodeDictionary(data []byte, pos int) (Value, int, error) {
	pos++
	dict := Dict{}
	var lastKey string

	for pos < len(data) && data[pos] != 'e' {
		tempKey, newPos, err := decodeValue(data, pos)
		if err != nil {
			return nil, pos, err
		}

		key, ok := tempKey.(String)
		if !ok {
			return nil, pos, fmt.Errorf("dict key must be string, got %T", tempKey)
		}
		pos = newPos

		currentKey := string(key)
		if lastKey != "" && currentKey <= lastKey {
			return nil, pos, fmt.Errorf("dict keys must be sorted, got '%s' after '%s'", currentKey, lastKey)
		}
		lastKey = currentKey

		val, newPos, err := decodeValue(data, pos)
		if err != nil {
			return nil, pos, err
		}

		dict[currentKey] = val
		pos = newPos
	}

	if pos >= len(data) || data[pos] != 'e' {
		return nil, pos, fmt.Errorf("dict not terminated with 'e'")
	}

	pos++
	return dict, pos, nil
}

func decodeValue(data []byte, pos int) (Value, int, error) {
	if pos >= len(data) {
		return nil, pos, fmt.Errorf("unexpected EOF")
	}

	switch data[pos] {
	case 'i':
		return decodeInteger(data, pos)
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return decodeString(data, pos)
	case 'l':
		return decodeList(data, pos)
	case 'd':
		return decodeDictionary(data, pos)
	default:
		return nil, pos, fmt.Errorf("invalid datatype: %c", data[pos])
	}
}

func Decode(data []byte) (Value, error) {
	val, pos, err := decodeValue(data, 0)
	if pos != len(data) {
		return nil, fmt.Errorf("invalid file")
	}
	return val, err
}
