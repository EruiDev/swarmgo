package bencode

import (
	"fmt"
	torr "torrent-client/torrent"
)

func getFieldAsType[T any](dict Dict, fieldName string) (T, error) {
	val, exists := dict[fieldName]
	if !exists {
		var zero T
		return zero, fmt.Errorf("%s field missing", fieldName)
	}
	typedVal, ok := val.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("incorrect type for %s: %T", fieldName, val)
	}
	return typedVal, nil
}

func unmarshalInfo(val Value) (torr.TorrentInfo, error) {
	dict, ok := val.(Dict)
	if !ok {
		return torr.TorrentInfo{}, fmt.Errorf("info should be a dictionary")
	}
	var info torr.TorrentInfo
	name, err := getFieldAsType[String](dict, "name")
	if err != nil {
		return info, err
	}
	info.Name = string(name)

	pieceLength, err := getFieldAsType[Integer](dict, "piece length")
	if err != nil {
		return info, err
	}
	info.PieceLength = int64(pieceLength)

	pieces, err := getFieldAsType[String](dict, "pieces")
	if err != nil {
		return info, err
	}
	info.Pieces = []byte(pieces)

	length, err := getFieldAsType[Integer](dict, "length")

	if err != nil {
		return info, err
	}

	info.Length = int64(length)
	return info, nil
}

func Unmarshal(val Value, torrent *torr.Torrent) error {
	dict, ok := val.(Dict)
	if !ok {
		return fmt.Errorf("torrent should start with a dictionary")
	}

	announceVal, exists := dict["announce"]
	if !exists {
		return fmt.Errorf("announce field missing")
	}
	announce, ok := announceVal.(String)
	if !ok {
		return fmt.Errorf("incorrect type for announce: %T", announceVal)
	}
	torrent.Announce = string(announce)

	infoVal, exists := dict["info"]
	if !exists {
		return fmt.Errorf("info field missing")
	}
	info, err := unmarshalInfo(infoVal)
	if err != nil {
		return err
	}
	torrent.Info = info

	return nil
}
