package download

import (
	"context"
	"os"
	"sync"
	torr "torrent-client/torrent"
	"torrent-client/tracker"
)

const (
	maxConcurrentBlocks = 30
	maxPeersToUse       = 50
)

type PieceWork struct {
	index  int
	hash   []byte
	length int
}

type PieceResult struct {
	index int
	data  []byte
	err   error
}

type workerConfig struct {
	peer      tracker.Peer
	infoHash  [20]byte
	peerID    [20]byte
	workQueue chan PieceWork
	results   chan<- PieceResult
	semaphore chan struct{}
	ctx       context.Context
}

type resultCollector struct {
	outputFile    *os.File
	torrent       torr.Torrent
	numPieces     int
	workQueue     chan PieceWork
	results       <-chan PieceResult
	downloaded    int
	downloadedMap map[int]bool
	failedPieces  map[int]PieceWork
	fileMutex     sync.Mutex
	failedMutex   sync.Mutex
}
