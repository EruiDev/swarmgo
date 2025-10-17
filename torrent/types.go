package torrent

type Torrent struct {
	Announce string
	Info     TorrentInfo
}

type TorrentInfo struct {
	Name        string
	PieceLength int64
	Pieces      []byte
	Length      int64
	Files       []File
}

type File struct {
	Name   string
	Length int64
}
