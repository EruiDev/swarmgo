package bencode

import (
	"fmt"
)

func parseInteger(data []byte, pos int) (Value, int, error) {
	pos++
	num, pos, err := getNumberTilStopper(data, pos, 'e')
	if err != nil {
		return nil, pos, err
	}
	return Integer(num), pos + 1, nil
}

func parseString(data []byte, pos int) (Value, int, error) {
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

func parseList(data []byte, pos int) (Value, int, error) {
	pos++
	list := List{}

	for pos < len(data) && data[pos] != 'e' {
		val, newPos, err := parseValue(data, pos)
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

func parseDictionary(data []byte, pos int) (Value, int, error) {
	pos++
	dict := Dict{}
	var lastKey string

	for pos < len(data) && data[pos] != 'e' {
		tempKey, newPos, err := parseValue(data, pos)
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

		val, newPos, err := parseValue(data, pos)
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

func parseValue(data []byte, pos int) (Value, int, error) {
	if pos >= len(data) {
		return nil, pos, fmt.Errorf("unexpected EOF")
	}

	switch data[pos] {
	case 'i':
		return parseInteger(data, pos)
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return parseString(data, pos)
	case 'l':
		return parseList(data, pos)
	case 'd':
		return parseDictionary(data, pos)
	default:
		return nil, pos, fmt.Errorf("invalid datatype: %c", data[pos])
	}
}

func Parse(data []byte) (Value, error) {
	val, pos, err := parseValue(data, 0)
	if pos != len(data) {
		return nil, fmt.Errorf("invalid file")
	}
	return val, err
}
