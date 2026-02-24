package download

import (
	"context"
	"encoding/binary"
	"fmt"
	"swarmgo/peers"
	"swarmgo/tracker"
	"time"
)

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

	ctx, cancel := context.WithCancel(cfg.ctx)
	defer cancel()
	go sendKeepalive(ctx, pc)

	downloadFromPeer(pc, cfg)

	return nil
}

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

func downloadBlock(pc *peers.PeerConnection, pieceIndex, offset, size int) ([]byte, error) {
	pc.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := pc.RequestPiece(pieceIndex, offset, size)
	if err != nil {
		return nil, err
	}

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
