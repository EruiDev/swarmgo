package tracker

type TrackerRequest struct {
	InfoHash   [20]byte
	PeerID     [20]byte
	Port       int
	Uploaded   int64
	Downloaded int64
	Left       int64
}

type TrackerResponse struct {
	Interval int64
	Peers    []Peer
}

type Peer struct {
	IP   string
	Port int
}
