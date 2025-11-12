package main

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"time"
	"torrent-client/bencode"
	"torrent-client/peers"
	torr "torrent-client/torrent"
	"torrent-client/tracker"
)

type PieceResult struct {
	index int
	data  []byte
	err   error
}

type PieceWork struct {
	index  int
	hash   []byte
	length int
}

func verifyPiece(data []byte, expectedHash []byte) bool {
	hash := sha1.Sum(data)
	return string(hash[:]) == string(expectedHash)
}

func main() {
	var torrent torr.Torrent
	torrentData, err := os.ReadFile("debian.torrent")

	if err != nil {
		fmt.Print(err)
		return
	}

	val, err := bencode.Decode(torrentData)
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
		return
	}

	// Calculate number of pieces
	numPieces := len(torrent.Info.Pieces) / 20
	fmt.Printf("Starting download of %d pieces (%d bytes total)\n", numPieces, torrent.Info.Length)

	// Create work queue
	workQueue := make(chan PieceWork, numPieces)
	for i := 0; i < numPieces; i++ {
		pieceLength := int(torrent.Info.PieceLength)
		// Last piece might be smaller
		if i == numPieces-1 {
			lastPieceLength := int(torrent.Info.Length) - i*int(torrent.Info.PieceLength)
			if lastPieceLength < pieceLength {
				pieceLength = lastPieceLength
			}
		}
		workQueue <- PieceWork{
			index:  i,
			hash:   torrent.Info.Pieces[i*20 : (i+1)*20],
			length: pieceLength,
		}
	}
	close(workQueue)

	// Create output file
	outputFile := torrent.Info.Name
	file, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer file.Close()

	// Truncate/pre-allocate file to full size
	err = file.Truncate(torrent.Info.Length)
	if err != nil {
		fmt.Printf("Failed to truncate file: %v\n", err)
		return
	}

	results := make(chan PieceResult, numPieces)
	var wg sync.WaitGroup

	maxConcurrent := 30
	semaphore := make(chan struct{}, maxConcurrent)

	maxPeers := 50
	peersToTry := len(res.Peers)
	if peersToTry > maxPeers {
		peersToTry = maxPeers
	}

	// Start worker goroutines
	for i := 0; i < peersToTry; i++ {
		peer := res.Peers[i]
		wg.Add(1)

		go func(p tracker.Peer) {
			defer wg.Done()

			pc, err := peers.Connect(p.IP, p.Port)
			if err != nil {
				fmt.Printf("[%s:%d] Connect failed: %v\n", p.IP, p.Port, err)
				return
			}
			defer pc.Conn.Close()

			pc.Conn.SetReadDeadline(time.Now().Add(15 * time.Second))
			pc.Conn.SetWriteDeadline(time.Now().Add(15 * time.Second))

			err = pc.Handshake(req.InfoHash, req.PeerID)
			if err != nil {
				fmt.Printf("[%s:%d] Handshake failed: %v\n", p.IP, p.Port, err)
				return
			}

			bitfieldMsg, err := pc.ReadMessage()
			if err != nil {
				fmt.Printf("[%s:%d] Bitfield read failed: %v\n", p.IP, p.Port, err)
				return
			}
			if bitfieldMsg.Type != peers.Bitfield {
				fmt.Printf("[%s:%d] Expected bitfield, got %d\n", p.IP, p.Port, bitfieldMsg.Type)
				return
			}

			fmt.Printf("[%s:%d] Connected successfully\n", p.IP, p.Port)

			interested := &peers.Message{Type: peers.Interested, Payload: []byte{}}
			pc.SendMessage(interested)

			gotUnchoke := false
			for i := 0; i < 50; i++ {
				msg, err := pc.ReadMessage()
				if err != nil {
					fmt.Printf("[%s:%d] Error reading unchoke: %v\n", p.IP, p.Port, err)
					break
				}
				if msg.Type == peers.Unchoke {
					gotUnchoke = true
					break
				}
			}

			if !gotUnchoke {
				fmt.Printf("[%s:%d] Did not get unchoke\n", p.IP, p.Port)
				return
			}

			fmt.Printf("[%s:%d] Unchoked, ready to download\n", p.IP, p.Port)

			// Download pieces from work queue
			for work := range workQueue {
				semaphore <- struct{}{}
				func(w PieceWork) {
					defer func() { <-semaphore }()

					blockSize := 16384 // 16KB
					pieceData := make([]byte, w.length)

					for offset := 0; offset < w.length; offset += blockSize {
						requestSize := blockSize
						if offset+blockSize > w.length {
							requestSize = w.length - offset
						}

						pc.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))
						err = pc.RequestPiece(w.index, offset, requestSize)
						if err != nil {
							fmt.Printf("[%s:%d] Request failed for piece %d: %v\n", p.IP, p.Port, w.index, err)
							results <- PieceResult{index: w.index, err: err}
							return
						}

						// Read the piece block
						gotBlock := false
						for attempt := 0; attempt < 20; attempt++ {
							msg, err := pc.ReadMessage()
							if err != nil {
								fmt.Printf("[%s:%d] Error reading block for piece %d at %d: %v\n", p.IP, p.Port, w.index, offset, err)
								break
							}

							if msg.Type == peers.Piece {
								pieceIdx := binary.BigEndian.Uint32(msg.Payload[0:4])
								blockOffset := binary.BigEndian.Uint32(msg.Payload[4:8])
								blockData := msg.Payload[8:]

								if int(pieceIdx) == w.index && int(blockOffset) == offset {
									copy(pieceData[offset:], blockData)
									gotBlock = true
									break
								}
							}
						}

						if !gotBlock {
							fmt.Printf("[%s:%d] Failed to get block for piece %d at offset %d\n", p.IP, p.Port, w.index, offset)
							results <- PieceResult{index: w.index, err: fmt.Errorf("block timeout")}
							return
						}
					}

					// Verify piece hash
					if verifyPiece(pieceData, w.hash) {
						results <- PieceResult{index: w.index, data: pieceData}
					} else {
						fmt.Printf("[%s:%d] Hash verification failed for piece %d\n", p.IP, p.Port, w.index)
						results <- PieceResult{index: w.index, err: fmt.Errorf("hash mismatch")}
					}
				}(work)
			}
		}(peer)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and write to file
	downloaded := 0
	failed := make(map[int]bool)
	var fileMutex sync.Mutex

	for result := range results {
		if result.err == nil && len(result.data) > 0 {
			fileMutex.Lock()
			offset := int64(result.index) * torrent.Info.PieceLength
			_, err = file.WriteAt(result.data, offset)
			fileMutex.Unlock()

			if err != nil {
				fmt.Printf("Failed to write piece %d: %v\n", result.index, err)
			} else {
				downloaded++
				fmt.Printf("Progress: %d/%d pieces (%.1f%%)\n", downloaded, numPieces, float64(downloaded)*100/float64(numPieces))
			}
		} else if result.err != nil {
			failed[result.index] = true
		}
	}

	if downloaded == numPieces {
		fmt.Printf("\n✓ Download complete! File saved as: %s\n", outputFile)
	} else {
		fmt.Printf("\n✗ Download incomplete: %d/%d pieces downloaded\n", downloaded, numPieces)
	}
}
