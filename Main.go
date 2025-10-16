package main

import (
	"fmt"
	"os"
	"torrent-client/bencode"
)

func main() {
	file, err := os.ReadFile("test.torrent")

	if err != nil {
		fmt.Print(err)
		return
	}

	test, err := bencode.Parse(file)
	if err != nil {
		fmt.Print(err)
		return
	}
	fmt.Print(test)
}
