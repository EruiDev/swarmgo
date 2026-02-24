# swarmgo

A concurrent BitTorrent client implementation written in Go that supports downloading files using the BitTorrent protocol.

## Features

- **Concurrent Piece Downloads**: Downloads multiple pieces simultaneously from different peers (up to 30 concurrent blocks)
- **Multi-Peer Support**: Connects to up to 50 peers concurrently for faster downloads
- **Automatic Retry**: Failed piece downloads are automatically retried
- **Progress Tracking**: Real-time download progress with percentage completion
- **Bencode Support**: Full implementation of BitTorrent's bencode encoding/decoding
- **Tracker Communication**: HTTP tracker support for peer discovery

## Architecture

The client is organized into several packages:

- `bencode/` - Bencode encoding/decoding and info hash calculation
- `torrent/` - Torrent file parsing and data structures
- `tracker/` - Tracker communication and peer discovery
- `peers/` - Peer protocol implementation and message handling
- `download/` - Download orchestration, piece management, and file writing

## Requirements

- Go 1.22.2 or higher

## Installation

```bash
# Clone the repository
git clone <repository-url>
cd swarmgo

# Build the binary
go build -o swarmgo
```

## Usage

```bash
./swarmgo <torrent-file>
```

Example:
```bash
./swarmgo debian-13.1.0-amd64-netinst.iso.torrent
```

The downloaded file will be saved with the name specified in the torrent file's metadata.

## How It Works

1. **Parse Torrent File**: Reads and decodes the `.torrent` file using bencode
2. **Contact Tracker**: Communicates with the tracker to discover peers
3. **Connect to Peers**: Establishes connections with available peers
4. **Download Pieces**: Downloads file pieces concurrently from multiple peers
5. **Verify & Write**: Verifies piece integrity using SHA-1 hashes and writes to disk
6. **Retry Failed Pieces**: Automatically retries any failed downloads

## Configuration

Current limits (defined in `download/types.go`):
- Maximum concurrent block downloads: 30
- Maximum peers to use: 50

## Project Structure

```
.
├── bencode/          # Bencode encoding/decoding
├── download/         # Download orchestration
│   ├── collector.go  # Result collection and file writing
│   ├── download.go   # Main download logic
│   ├── file.go       # File preparation
│   ├── types.go      # Type definitions
│   ├── verify.go     # Piece verification
│   └── worker.go     # Peer worker implementation
├── peers/            # Peer protocol
│   ├── connection.go # Peer connection handling
│   ├── messages.go   # BitTorrent messages
│   └── types.go      # Peer types
├── torrent/          # Torrent file handling
│   ├── types.go      # Torrent structures
│   └── utils.go      # Utility functions
├── tracker/          # Tracker communication
│   ├── contact.go    # Tracker HTTP requests
│   └── types.go      # Tracker types
└── main.go           # Entry point
```

## Technical Details

### Concurrent Design

- Each peer runs in its own goroutine
- Work queue distributes pieces to available workers
- Results channel collects completed pieces
- Semaphore controls maximum concurrent block requests

### Piece Verification

All downloaded pieces are verified against SHA-1 hashes from the torrent metadata before being written to disk.

## Limitations

- Currently supports single-file torrents only
- HTTP trackers only (no UDP tracker support)
- No DHT (Distributed Hash Table) support
- No peer exchange (PEX) support
- No magnet link support

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

[Add your license here]
