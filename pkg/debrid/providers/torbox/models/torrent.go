package models

import (
	"encoding/json"
	"time"
)

type Torrent struct {
	Id               int          `json:"id"`
	AuthId           string       `json:"auth_id"`
	Server           int          `json:"server"`
	Hash             string       `json:"hash"`
	Name             string       `json:"name"`
	Magnet           any          `json:"magnet"`
	Size             int64        `json:"size"`
	Active           bool         `json:"active"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
	DownloadState    string       `json:"download_state"`
	Seeds            int          `json:"seeds"`
	Peers            int          `json:"peers"`
	Ratio            float64      `json:"ratio"`
	Progress         float64      `json:"progress"`
	DownloadSpeed    int64        `json:"download_speed"`
	UploadSpeed      int          `json:"upload_speed"`
	ETA              int          `json:"eta"`
	TorrentFile      bool         `json:"torrent_file"`
	ExpiresAt        any          `json:"expires_at"`
	DownloadPresent  bool         `json:"download_present"`
	Files            []TorboxFile `json:"files"`
	DownloadPath     string       `json:"download_path"`
	InactiveCheck    int          `json:"inactive_check"`
	Availability     float64      `json:"availability"`
	DownloadFinished bool         `json:"download_finished"`
	Tracker          any          `json:"tracker"`
	TotalUploaded    int          `json:"total_uploaded"`
	TotalDownloaded  int          `json:"total_downloaded"`
	Cached           bool         `json:"cached"`
	Owner            string       `json:"owner"`
	SeedTorrent      bool         `json:"seed_torrent"`
	AllowZipped      bool         `json:"allow_zipped"`
	LongTermSeeding  bool         `json:"long_term_seeding"`
	TrackerMessage   any          `json:"tracker_message"`
}

func (t *Torrent) UnmarshalJSON(d []byte) error {
	type Alias Torrent
	type Aux struct {
		*Alias

		TorrentID *int `json:"torrent_id"`
		QueuedID  *int `json:"queued_id"`
	}

	aux := &Aux{
		Alias: (*Alias)(t),
	}

	err := json.Unmarshal(d, &aux)
	if err != nil {
		return err
	}

	if t.Id == 0 {
		if aux.TorrentID != nil {
			t.Id = *aux.TorrentID
		}

		if aux.QueuedID != nil {
			t.Id = *aux.QueuedID
		}
	}

	return err
}
