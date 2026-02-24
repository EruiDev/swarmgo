package download

import (
	"context"
	"fmt"
	"sync"
	torr "torrent-client/torrent"
	"torrent-client/tracker"
)

func Download(file string) error {
	torrent, req, res, err := initializeDownload(file)
	if err != nil {
		return err
	}

	outputFile, err := PrepareFile(torrent.Info.Name, torrent.Info.Length)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	numPieces := len(torrent.Info.Pieces) / 20
	workQueue := createQueue(numPieces, *torrent)
	results := make(chan PieceResult, numPieces)

	fmt.Printf("Starting download of %d pieces (%d bytes total)\n", numPieces, torrent.Info.Length)

	ctx := context.Background()
	peersToTry := min(len(res.Peers), maxPeersToUse)
	startPeerWorkers(ctx, res.Peers[:peersToTry], *req, workQueue, results)

	collector := newResultCollector(outputFile, *torrent, numPieces, workQueue, results)
	collector.collect()
	collector.printSummary()

	return nil
}

func initializeDownload(file string) (*torr.Torrent, *tracker.TrackerRequest, *tracker.TrackerResponse, error) {
	var torrent torr.Torrent
	req, err := ReadAndExtractFile(file, &torrent)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read torrent file: %w", err)
	}

	res, err := tracker.ContactTracker(torrent.Announce, req)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to contact tracker: %w", err)
	}

	return &torrent, &req, &res, nil
}

func startPeerWorkers(ctx context.Context, peers []tracker.Peer, req tracker.TrackerRequest, workQueue chan PieceWork, results chan<- PieceResult) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrentBlocks)

	for _, peer := range peers {
		wg.Add(1)
		go func(p tracker.Peer) {
			defer wg.Done()

			cfg := workerConfig{
				peer:      p,
				infoHash:  req.InfoHash,
				peerID:    req.PeerID,
				workQueue: workQueue,
				results:   results,
				semaphore: semaphore,
				ctx:       ctx,
			}

			if err := startPeerWorker(cfg); err != nil {
				return
			}
		}(peer)
	}

	go func() {
		wg.Wait()
		close(results)
	}()
}

func createQueue(numPieces int, torrent torr.Torrent) chan PieceWork {
	workQueue := make(chan PieceWork, numPieces*2)
	for i := range numPieces {
		pieceLength := int(torrent.Info.PieceLength)
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
	return workQueue
}
