package download

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"
	"torrent-client/peers"
	"torrent-client/tracker"
)

// workerConfig holds configuration for a download worker
type workerConfig struct {
	peer       tracker.Peer
	infoHash   [20]byte
	peerID     [20]byte
	workQueue  chan PieceWork
	results    chan<- PieceResult
	semaphore  chan struct{}
	ctx        context.Context
}

// startPeerWorker manages a connection to a single peer and downloads pieces
func startPeerWorker(cfg workerConfig) error {
	pc, err := connectAndHandshake(cfg.peer, cfg.infoHash, cfg.peerID)
	if err != nil {
		return err
	}
	defer pc.Conn.Close()

	if err := waitForUnchoke(pc, cfg.peer); err != nil {
		return err
	}

	fmt.Printf("[%s:%d] Unchoked, ready to download\n", cfg.peer.IP, cfg.peer.Port)

	// Start keepalive sender
	ctx, cancel := context.WithCancel(cfg.ctx)
	defer cancel()
	go sendKeepalive(ctx, pc)

	// Download pieces from work queue
	downloadFromPeer(pc, cfg)

	return nil
}

// connectAndHandshake establishes connection and performs handshake with a peer
func connectAndHandshake(peer tracker.Peer, infoHash, peerID [20]byte) (*peers.PeerConnection, error) {
	pc, err := peers.Connect(peer.IP, peer.Port)
	if err != nil {
		return nil, fmt.Errorf("[%s:%d] connect failed: %w", peer.IP, peer.Port, err)
	}

	pc.Conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	pc.Conn.SetWriteDeadline(time.Now().Add(15 * time.Second))

	err = pc.Handshake(infoHash, peerID)
	if err != nil {
		pc.Conn.Close()
		return nil, fmt.Errorf("[%s:%d] handshake failed: %w", peer.IP, peer.Port, err)
	}

	// Read bitfield
	pc.Conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	bitfieldMsg, err := pc.ReadMessage()
	if err != nil {
		pc.Conn.Close()
		return nil, fmt.Errorf("[%s:%d] bitfield read failed: %w", peer.IP, peer.Port, err)
	}

	if bitfieldMsg.Type != peers.Bitfield {
		pc.Conn.Close()
		return nil, fmt.Errorf("[%s:%d] expected bitfield, got %d", peer.IP, peer.Port, bitfieldMsg.Type)
	}

	fmt.Printf("[%s:%d] Connected successfully\n", peer.IP, peer.Port)
	return pc, nil
}

// waitForUnchoke sends interested message and waits for unchoke
func waitForUnchoke(pc *peers.PeerConnection, peer tracker.Peer) error {
	interested := &peers.Message{Type: peers.Interested, Payload: []byte{}}
	pc.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := pc.SendMessage(interested); err != nil {
		return fmt.Errorf("[%s:%d] failed to send interested: %w", peer.IP, peer.Port, err)
	}

	for i := 0; i < 50; i++ {
		pc.Conn.SetReadDeadline(time.Now().Add(15 * time.Second))
		msg, err := pc.ReadMessage()
		if err != nil {
			return fmt.Errorf("[%s:%d] error reading unchoke: %w", peer.IP, peer.Port, err)
		}
		if msg.Type == peers.Unchoke {
			return nil
		}
	}

	return fmt.Errorf("[%s:%d] did not get unchoke", peer.IP, peer.Port)
}

// sendKeepalive sends periodic keepalive messages to maintain connection
func sendKeepalive(ctx context.Context, pc *peers.PeerConnection) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pc.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			pc.Conn.Write([]byte{0, 0, 0, 0})
		case <-ctx.Done():
			return
		}
	}
}

// downloadFromPeer downloads pieces from the work queue using the peer connection
func downloadFromPeer(pc *peers.PeerConnection, cfg workerConfig) {
	connectionBroken := false

	for work := range cfg.workQueue {
		if connectionBroken {
			select {
			case cfg.workQueue <- work:
			default:
			}
			break
		}

		cfg.semaphore <- struct{}{}
		func(w PieceWork) {
			defer func() { <-cfg.semaphore }()

			pieceData, err := downloadPiece(pc, w)
			if err != nil {
				connectionBroken = true
				cfg.results <- PieceResult{index: w.index, err: err}
				return
			}

			// Verify piece hash
			if verifyPiece(pieceData, w.hash) {
				cfg.results <- PieceResult{index: w.index, data: pieceData}
			} else {
				cfg.results <- PieceResult{index: w.index, err: fmt.Errorf("hash mismatch")}
			}
		}(work)
	}
}

// downloadPiece downloads a complete piece by requesting blocks
func downloadPiece(pc *peers.PeerConnection, work PieceWork) ([]byte, error) {
	blockSize := 16384 // 16KB
	pieceData := make([]byte, work.length)

	for offset := 0; offset < work.length; offset += blockSize {
		requestSize := blockSize
		if offset+blockSize > work.length {
			requestSize = work.length - offset
		}

		blockData, err := downloadBlock(pc, work.index, offset, requestSize)
		if err != nil {
			return nil, err
		}

		copy(pieceData[offset:], blockData)
	}

	return pieceData, nil
}

// downloadBlock requests and receives a single block from a peer
func downloadBlock(pc *peers.PeerConnection, pieceIndex, offset, size int) ([]byte, error) {
	pc.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := pc.RequestPiece(pieceIndex, offset, size)
	if err != nil {
		return nil, err
	}

	// Read the piece block
	for attempt := 0; attempt < 20; attempt++ {
		pc.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		msg, err := pc.ReadMessage()
		if err != nil {
			return nil, err
		}

		if msg.Type == peers.Piece {
			pieceIdx := binary.BigEndian.Uint32(msg.Payload[0:4])
			blockOffset := binary.BigEndian.Uint32(msg.Payload[4:8])
			blockData := msg.Payload[8:]

			if int(pieceIdx) == pieceIndex && int(blockOffset) == offset {
				return blockData, nil
			}
		}
	}

	return nil, fmt.Errorf("block timeout")
}
