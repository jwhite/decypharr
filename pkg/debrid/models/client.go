package models

import "time"

type ClientOptions struct {
	Name      string
	MountPath string
	Profile   *Profile
	Host      string
	APIKey    string

	DownloadUncached     bool
	CheckCached          bool
	AddSamples           bool
	AutoExpireLinksAfter time.Duration
	DownloadingStatus    []string
}

type Client interface {
	ClientOptions() ClientOptions

	GetAccounts() *Accounts
	SyncAccounts() error
	GetAvailableSlots() (*int, error)

	GetTorrent(id string) (*DebridTorrent, error)
	GetTorrents() ([]DebridTorrent, error)
	CreateTorrent(tr *DebridTorrent) (*DebridTorrent, error)
	UpdateTorrent(torrent *DebridTorrent) error
	DeleteTorrent(id string) error

	GetTorrentAvailable(hashes []string) map[string]bool
	GetTorrentStatus(tr *DebridTorrent) (*DebridTorrent, error)

	GetDownloadLink(tr *DebridTorrent, file *File) (*DownloadLink, error)
	GetDownloadLinks() (map[string]*DownloadLink, error)
	GetFileDownloadLinks(tr *DebridTorrent) error
	CheckDownloadLink(link string) error
	DeleteDownloadLink(linkId string) error
}
