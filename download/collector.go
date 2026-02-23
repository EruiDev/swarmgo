package download

import (
	"fmt"
	"os"
	"sync"
	torr "torrent-client/torrent"
)

// resultCollector manages the collection and writing of downloaded pieces
type resultCollector struct {
	outputFile      *os.File
	torrent         torr.Torrent
	numPieces       int
	workQueue       chan PieceWork
	results         <-chan PieceResult
	downloaded      int
	downloadedMap   map[int]bool
	failedPieces    map[int]PieceWork
	fileMutex       sync.Mutex
	failedMutex     sync.Mutex
}

// newResultCollector creates a new result collector
func newResultCollector(file *os.File, torrent torr.Torrent, numPieces int, workQueue chan PieceWork, results <-chan PieceResult) *resultCollector {
	return &resultCollector{
		outputFile:    file,
		torrent:       torrent,
		numPieces:     numPieces,
		workQueue:     workQueue,
		results:       results,
		downloadedMap: make(map[int]bool),
		failedPieces:  make(map[int]PieceWork),
	}
}

// collect processes download results and writes them to file
func (rc *resultCollector) collect() {
	for result := range rc.results {
		if result.err == nil && len(result.data) > 0 {
			rc.handleSuccessfulPiece(result)
		} else if result.err != nil {
			rc.handleFailedPiece(result)
		}
	}
}

// handleSuccessfulPiece writes a successfully downloaded piece to file
func (rc *resultCollector) handleSuccessfulPiece(result PieceResult) {
	rc.fileMutex.Lock()
	offset := int64(result.index) * rc.torrent.Info.PieceLength
	_, err := rc.outputFile.WriteAt(result.data, offset)
	rc.fileMutex.Unlock()

	if err != nil {
		fmt.Printf("Failed to write piece %d: %v\n", result.index, err)
		return
	}

	rc.downloaded++
	rc.downloadedMap[result.index] = true

	// Remove from failed pieces if it was there
	rc.failedMutex.Lock()
	delete(rc.failedPieces, result.index)
	rc.failedMutex.Unlock()

	fmt.Printf("Progress: %d/%d pieces (%.1f%%)\n",
		rc.downloaded, rc.numPieces,
		float64(rc.downloaded)*100/float64(rc.numPieces))
}

// handleFailedPiece attempts to retry a failed piece download
func (rc *resultCollector) handleFailedPiece(result PieceResult) {
	rc.failedMutex.Lock()
	defer rc.failedMutex.Unlock()

	if rc.downloadedMap[result.index] {
		return
	}

	// Create work item for retry
	work := rc.createPieceWork(result.index)
	rc.failedPieces[result.index] = work

	// Immediately retry failed piece by putting it back in queue
	select {
	case rc.workQueue <- work:
	default:
		// Queue full, will retry in next pass
	}
}

// createPieceWork creates a PieceWork item for a given piece index
func (rc *resultCollector) createPieceWork(index int) PieceWork {
	pieceLength := int(rc.torrent.Info.PieceLength)

	// Last piece might be smaller
	if index == rc.numPieces-1 {
		lastPieceLength := int(rc.torrent.Info.Length) - index*int(rc.torrent.Info.PieceLength)
		if lastPieceLength < pieceLength {
			pieceLength = lastPieceLength
		}
	}

	return PieceWork{
		index:  index,
		hash:   rc.torrent.Info.Pieces[index*20 : (index+1)*20],
		length: pieceLength,
	}
}

// printSummary prints the final download summary
func (rc *resultCollector) printSummary() {
	if rc.downloaded == rc.numPieces {
		fmt.Printf("\n✓ Download complete! File saved as: %s\n", rc.torrent.Info.Name)
	} else {
		fmt.Printf("\n✗ Download incomplete: %d/%d pieces downloaded (%d failed)\n",
			rc.downloaded, rc.numPieces, len(rc.failedPieces))
		if len(rc.failedPieces) > 0 {
			fmt.Printf("Failed pieces: ")
			for idx := range rc.failedPieces {
				fmt.Printf("%d ", idx)
			}
			fmt.Println()
		}
	}
}
