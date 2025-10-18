package main

import (
	"fmt"
	"os"
	"torrent-client/bencode"
	"torrent-client/torrent"
	torr "torrent-client/torrent"
	"torrent-client/tracker"
)

func main() {
	var torrent torrent.Torrent
	file, err := os.ReadFile("debian.torrent")

	if err != nil {
		fmt.Print(err)
		return
	}

	val, err := bencode.Decode(file)
	if err != nil {
		fmt.Print(err)
		return
	}
	err = bencode.Unmarshal(val, &torrent)
	if err != nil {
		fmt.Print(err)
		return
	}
	hash, err := bencode.GetInfoHash(val)
	if err != nil {
		fmt.Print(err)
		return
	}

	peerID, err := torr.GeneratePeerID()
	if err != nil {
		fmt.Print(err)
		return
	}
	req := tracker.TrackerRequest{
		InfoHash:   hash,
		PeerID:     peerID,
		Port:       6881,
		Uploaded:   0,
		Downloaded: 0,
		Left:       torrent.Info.Length,
	}

	res, err := tracker.ContactTracker(torrent.Announce, req)
	if err != nil {
		fmt.Print(err)
	}
	fmt.Print(res)
}
