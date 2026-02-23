package main

import (
	"fmt"
	"os"
	"torrent-client/download"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: torrent-client <torrent-file>")
		os.Exit(1)
	}

	err := download.Download(os.Args[1])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
