package tracker

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"torrent-client/bencode"
)

func ContactTracker(announceURL string, req TrackerRequest) (TrackerResponse, error) {
	link := url.Values{}
	link.Set("info_hash", string(req.InfoHash[:]))
	link.Set("peer_id", string(req.PeerID[:]))
	link.Set("port", strconv.FormatInt(int64(req.Port), 10))
	link.Set("uploaded", strconv.FormatInt(req.Uploaded, 10))
	link.Set("downloaded", strconv.FormatInt(req.Downloaded, 10))
	link.Set("left", strconv.FormatInt(req.Left, 10))
	link.Set("compact", "1")

	fullURL := announceURL + "?" + link.Encode()

	res, err := http.Get(fullURL)
	if err != nil {
		return TrackerResponse{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return TrackerResponse{}, fmt.Errorf("request failed with code: %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return TrackerResponse{}, err
	}

	decodeBody, err := bencode.Decode(body)
	if err != nil {
		return TrackerResponse{}, err
	}

	intervalVal, exists := decodeBody.(bencode.Dict)["interval"]
	if !exists {
		return TrackerResponse{}, fmt.Errorf("interval not found")
	}
	interval, ok := intervalVal.(bencode.Integer)
	if !ok {
		return TrackerResponse{}, fmt.Errorf("peers has incorrect type")
	}

	peersVal, exists := decodeBody.(bencode.Dict)["peers"]
	if !exists {
		return TrackerResponse{}, fmt.Errorf("peers not found")
	}
	peers, ok := peersVal.(bencode.String)
	if !ok {
		return TrackerResponse{}, fmt.Errorf("peers has incorrect type")
	}
	peerList, _ := peerParser([]byte(peers))
	final := TrackerResponse{
		Interval: int64(interval),
		Peers:    peerList,
	}
	return final, nil
}

func peerParser(data []byte) ([]Peer, error) {
	if len(data)%6 != 0 {
		return nil, fmt.Errorf("invalid amount of bytes")
	}

	list := []Peer{}
	for i := 0; i+6 <= len(data); i = 6 + i {
		ip := fmt.Sprintf("%d.%d.%d.%d", data[i], data[i+1], data[i+2], data[i+3])
		port := int(data[i+4])<<8 | int(data[i+5])
		list = append(list, Peer{IP: ip, Port: port})
	}
	return list, nil
}
