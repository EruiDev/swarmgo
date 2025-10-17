package main

import (
	"fmt"
	"os"
	"torrent-client/bencode"
	"torrent-client/torrent"
)

func main() {
	file, err := os.ReadFile("debian.torrent")
	var torrent torrent.Torrent

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
	fmt.Print(hash)
}
