package bencode

import (
	"fmt"
	"sort"
)

func encodeInteger(v Integer) []byte {
	return []byte(fmt.Sprintf("i%de", v))
}

func encodeString(v String) []byte {
	return []byte(fmt.Sprintf("%d:%s", len(v), v))
}

func encodeList(v List) ([]byte, error) {
	var res []byte

	res = append(res, 'l')

	for _, val := range v {
		encoded, err := Encode(val)
		if err != nil {
			return nil, err
		}
		res = append(res, encoded...)
	}
	res = append(res, 'e')
	return res, nil
}

func encodeDictionary(v Dict) ([]byte, error) {
	var res []byte

	res = append(res, 'd')

	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		keyEncoded := encodeString(String(k))
		res = append(res, keyEncoded...)

		valEncoded, err := Encode(v[k])
		if err != nil {
			return nil, err
		}
		res = append(res, valEncoded...)
	}

	res = append(res, 'e')
	return res, nil
}

func Encode(val Value) ([]byte, error) {
	switch v := val.(type) {
	case List:
		return encodeList(v)
	case Dict:
		return encodeDictionary(v)
	case String:
		return encodeString(v), nil
	case Integer:
		return encodeInteger(v), nil
	default:
		return nil, fmt.Errorf("invalid type %T", v)
	}
}
