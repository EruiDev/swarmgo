package download

import (
	"fmt"
	"os"
	"torrent-client/bencode"
	torr "torrent-client/torrent"
	"torrent-client/tracker"
)

func ReadAndExtractFile(file string, torrent *torr.Torrent) (tracker.TrackerRequest, error) {
	torrentData, err := os.ReadFile(file)

	if err != nil {
		return tracker.TrackerRequest{}, err
	}

	val, err := bencode.Decode(torrentData)
	if err != nil {
		return tracker.TrackerRequest{}, err
	}
	err = bencode.Unmarshal(val, torrent)
	if err != nil {
		fmt.Print(err)
		return tracker.TrackerRequest{}, err
	}
	hash, err := bencode.GetInfoHash(val)
	if err != nil {
		fmt.Print(err)
		return tracker.TrackerRequest{}, err
	}

	peerID, err := torr.GeneratePeerID()
	if err != nil {
		fmt.Print(err)
		return tracker.TrackerRequest{}, err
	}
	req := tracker.TrackerRequest{
		InfoHash:   hash,
		PeerID:     peerID,
		Port:       6881,
		Uploaded:   0,
		Downloaded: 0,
		Left:       torrent.Info.Length,
	}
	return req, nil
}

func PrepareFile(name string, len int64) (*os.File, error) {
	outputFile := name
	file, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return &os.File{}, fmt.Errorf("Failed to create file: %w\n", err)
	}

	err = file.Truncate(len)
	if err != nil {
		return &os.File{}, fmt.Errorf("Failed to truncate file: %w\n", err)
	}
	return file, nil
}
