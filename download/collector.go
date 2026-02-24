package download

import (
	"fmt"
	"os"
	torr "swarmgo/torrent"
)

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

func (rc *resultCollector) collect() {
	for result := range rc.results {
		if result.err == nil && len(result.data) > 0 {
			rc.handleSuccessfulPiece(result)
		} else if result.err != nil {
			rc.handleFailedPiece(result)
		}
	}
}

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

	rc.failedMutex.Lock()
	delete(rc.failedPieces, result.index)
	rc.failedMutex.Unlock()

	fmt.Printf("Progress: %d/%d pieces (%.1f%%)\n",
		rc.downloaded, rc.numPieces,
		float64(rc.downloaded)*100/float64(rc.numPieces))
}

func (rc *resultCollector) handleFailedPiece(result PieceResult) {
	rc.failedMutex.Lock()
	defer rc.failedMutex.Unlock()

	if rc.downloadedMap[result.index] {
		return
	}

	work := rc.createPieceWork(result.index)
	rc.failedPieces[result.index] = work

	select {
	case rc.workQueue <- work:
	default:
	}
}

func (rc *resultCollector) createPieceWork(index int) PieceWork {
	pieceLength := int(rc.torrent.Info.PieceLength)

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

func (rc *resultCollector) printSummary() {
	if rc.downloaded == rc.numPieces {
		fmt.Printf("\nDownload complete! File saved as: %s\n", rc.torrent.Info.Name)
	} else {
		fmt.Printf("\nDownload incomplete: %d/%d pieces downloaded (%d failed)\n",
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
