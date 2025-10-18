package main

import (
	"fmt"
	"os"
	"torrent-client/bencode"
	"torrent-client/peers"
	torr "torrent-client/torrent"
	"torrent-client/tracker"
)

func main() {
	var torrent torr.Torrent
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
	for _, peer := range res.Peers {
		pc, err := peers.Connect(peer.IP, peer.Port)
		if err != nil {
			fmt.Printf("Failed to connect to %s:%d: %v\n", peer.IP, peer.Port, err)
			continue
		}
		defer pc.Conn.Close()

		err = pc.Handshake(req.InfoHash, req.PeerID)
		if err != nil {
			fmt.Printf("Failed to handshake to %s:%d: %v\n", peer.IP, peer.Port, err)
			continue
		}

		fmt.Printf("Connected successfully to %s:%d\n", peer.IP, peer.Port)
	}
}
