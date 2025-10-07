package store

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/sirrobot01/decypharr/internal/utils"
	debridTypes "github.com/sirrobot01/decypharr/pkg/debrid"
	"github.com/sirrobot01/decypharr/pkg/debrid/types"
)

func (s *Store) AddTorrent(ctx context.Context, importReq *ImportRequest) error {
	torrent := createTorrentFromMagnet(importReq)
	debridTorrent, err := debridTypes.Process(ctx, s.debrid, importReq.SelectedDebrid, importReq.Magnet, importReq.Arr, importReq.Action, importReq.DownloadUncached)

	if err != nil {
		var httpErr *utils.HTTPError
		if ok := errors.As(err, &httpErr); ok {
			switch httpErr.Code {
			case "too_many_active_downloads":
				// Handle too many active downloads error
				s.logger.Warn().Msgf("Too many active downloads for %s, adding to queue", importReq.Magnet.Name)

				if err := s.addToQueue(importReq); err != nil {
					s.logger.Error().Err(err).Msgf("Failed to add %s to queue", importReq.Magnet.Name)
					return err
				}

				torrent.State = "queued"
			default:
				// Unhandled error, return it, caller logs it
				return err
			}
		} else {
			// Unhandled error, return it, caller logs it
			return err
		}
	}
	torrent = s.partialTorrentUpdate(torrent, debridTorrent)
	s.torrents.AddOrUpdate(torrent)
	go s.processFiles(torrent, debridTorrent, importReq) // We can send async for file processing not to delay the response
	return nil
}

func (s *Store) processFiles(torrent *Torrent, debridTorrent *types.Torrent, importReq *ImportRequest) {
	if debridTorrent == nil {
		return
	}

	deb := s.debrid.Debrid(debridTorrent.Debrid)
	client := deb.Client()
	downloadingStatuses := client.GetDownloadingStatus()
	_arr := importReq.Arr

	backoff := time.NewTimer(s.refreshInterval)
	defer backoff.Stop()
	for debridTorrent.Status != "downloaded" {
		s.logger.Debug().Msgf("%s <- (%s) Download Progress: %.2f%%", debridTorrent.Debrid, debridTorrent.Name, debridTorrent.Progress)
		dbT, err := client.CheckStatus(debridTorrent)
		if err != nil {
			s.logger.Error().
				Str("torrent_id", debridTorrent.Id).
				Str("torrent_name", debridTorrent.Name).
				Err(err).
				Msg("Error checking torrent status")

			if dbT != nil && dbT.Id != "" {
				// Delete the torrent if it was not downloaded
				go func() {
					_ = client.DeleteTorrent(dbT.Id)
				}()
			}
			s.logger.Error().Msgf("Error checking status: %v", err)
			s.markAsFailed(torrent)

			go func() {
				_arr.Refresh()
			}()

			importReq.markAsFailed(err, torrent, debridTorrent)
			return
		}

		debridTorrent = dbT
		torrent = s.partialTorrentUpdate(torrent, debridTorrent)

		// Exit the loop for downloading statuses to prevent memory buildup
		if debridTorrent.Status == "downloaded" || !utils.Contains(downloadingStatuses, debridTorrent.Status) {
			break
		}

		<-backoff.C
		// Increase interval gradually, cap at max
		nextInterval := min(s.refreshInterval*2, 30*time.Second)
		backoff.Reset(nextInterval)
	}

	var torrentPath string
	var err error
	debridTorrent.Arr = _arr

	timer := time.Now()

	switch importReq.Action {
	case "symlink":
		s.logger.Debug().Msgf("Post-Download Action: Symlink")
		cache := deb.Cache()
		if cache != nil {
			s.logger.Info().Msgf("Using internal webdav for %s", debridTorrent.Debrid)
			err := cache.Add(debridTorrent)
			if err != nil {
				s.onFailed(err, torrent, debridTorrent, importReq)
				return
			}

			rclonePath := filepath.Join(debridTorrent.MountPath, cache.GetTorrentFolder(debridTorrent))
			torrentFolderNoExt := utils.RemoveExtension(debridTorrent.Name)
			torrentPath, err = s.createSymlinksWebdav(torrent, debridTorrent, rclonePath, torrentFolderNoExt)
			if err != nil {
				s.onFailed(err, torrent, debridTorrent, importReq)
				return
			}
		} else {
			torrentPath, err = s.processSymlink(torrent, debridTorrent)
		}

		if err != nil {
			s.onFailed(err, torrent, debridTorrent, importReq)
			return
		}

		if torrentPath == "" {
			err = fmt.Errorf("symlink path is empty for %s", debridTorrent.Name)
			s.onFailed(err, torrent, debridTorrent, importReq)
		}

		torrent.TorrentPath = torrentPath

		s.onSuccess(torrent, debridTorrent, importReq, timer)
		return
	case "download":
		s.logger.Debug().Msgf("Post-Download Action: Download")
		err := client.GetFileDownloadLinks(debridTorrent)
		if err != nil {
			s.onFailed(err, torrent, debridTorrent, importReq)
			return
		}

		torrentPath, err = s.processDownload(torrent, debridTorrent)
		if err != nil {
			s.onFailed(err, torrent, debridTorrent, importReq)

			return
		}

		if torrentPath == "" {
			err = fmt.Errorf("download path is empty for %s", debridTorrent.Name)
			s.onFailed(err, torrent, debridTorrent, importReq)

			return
		}

		torrent.TorrentPath = torrentPath

		s.onSuccess(torrent, debridTorrent, importReq, timer)
	case "none":
		s.logger.Debug().Msgf("Post-Download Action: None")
		torrent.TorrentPath = torrentPath

		s.onSuccess(torrent, debridTorrent, importReq, timer)
	default:
		// Action is none, do nothing, fallthrough
	}
}

func (s *Store) onSuccess(torrent *Torrent, debridTorrent *types.Torrent, importReq *ImportRequest, timer time.Time) {
	s.updateTorrent(torrent, debridTorrent)
	s.logger.Info().Msgf("Adding %s took %s", debridTorrent.Name, time.Since(timer))

	go importReq.markAsCompleted(torrent, debridTorrent)

	go func() {
		importReq.Arr.Refresh()
	}()

	go func() {
		deb := s.debrid.Debrid(debridTorrent.Debrid)
		debridClient := deb.Client()
		if debridClient == nil {
			return
		}

		err := debridClient.DeleteTorrent(debridTorrent.Id)
		if err != nil {
			s.logger.Warn().Err(err).Msgf("failed to delete torrent %s", debridTorrent.Id)
		}
	}()
}

func (s *Store) onFailed(err error, torrent *Torrent, debridTorrent *types.Torrent, importReq *ImportRequest) {
	s.markAsFailed(torrent)

	go func() {
		deb := s.debrid.Debrid(debridTorrent.Debrid)
		debridClient := deb.Client()
		if debridClient == nil {
			return
		}

		err := debridClient.DeleteTorrent(debridTorrent.Id)
		if err != nil {
			s.logger.Warn().Err(err).Msgf("failed to delete torrent %s", debridTorrent.Id)
		}
	}()

	s.logger.Error().Err(err).Msgf("error occured while processing torrent %s", debridTorrent.Name)
	importReq.markAsFailed(err, torrent, debridTorrent)
}

func (s *Store) markAsFailed(t *Torrent) *Torrent {
	t.State = "error"
	s.torrents.AddOrUpdate(t)

	return t
}

func (s *Store) partialTorrentUpdate(t *Torrent, debridTorrent *types.Torrent) *Torrent {
	if debridTorrent == nil {
		return t
	}

	addedOn, err := time.Parse(time.RFC3339, debridTorrent.Added)
	if err != nil {
		addedOn = time.Now()
	}

	totalSize := debridTorrent.Bytes
	progress := (cmp.Or(debridTorrent.Progress, 0.0)) / 100.0
	if math.IsNaN(progress) || math.IsInf(progress, 0) {
		progress = 0
	}

	sizeCompleted := int64(float64(totalSize) * progress)

	var speed int64
	if debridTorrent.Speed != 0 {
		speed = debridTorrent.Speed
	}

	var eta int
	if speed != 0 {
		eta = int((totalSize - sizeCompleted) / speed)
	}

	files := make([]*File, 0, len(debridTorrent.Files))
	for index, file := range debridTorrent.GetFiles() {
		files = append(files, &File{
			Index: index,
			Name:  file.Path,
			Size:  file.Size,
		})
	}

	t.DebridID = debridTorrent.Id
	t.Name = debridTorrent.Name
	t.AddedOn = addedOn.Unix()
	t.Files = files
	t.Debrid = debridTorrent.Debrid
	t.Size = totalSize
	t.Completed = sizeCompleted
	t.NumSeeds = debridTorrent.Seeders
	t.Downloaded = sizeCompleted
	t.DownloadedSession = sizeCompleted
	t.Uploaded = sizeCompleted
	t.UploadedSession = sizeCompleted
	t.AmountLeft = totalSize - sizeCompleted
	t.Progress = progress
	t.Eta = eta
	t.Dlspeed = speed
	t.Upspeed = speed
	t.ContentPath = filepath.Join(t.SavePath, t.Name) + string(os.PathSeparator)

	return t
}

func (s *Store) updateTorrent(t *Torrent, debridTorrent *types.Torrent) *Torrent {
	if debridTorrent == nil {
		return t
	}

	if debridClient := s.debrid.Clients()[debridTorrent.Debrid]; debridClient != nil {
		if debridTorrent.Status != "downloaded" {
			_ = debridClient.UpdateTorrent(debridTorrent)
		}
	}

	t = s.partialTorrentUpdate(t, debridTorrent)
	t.ContentPath = t.TorrentPath + string(os.PathSeparator)

	if t.IsReady() {
		t.State = "pausedUP"
		s.torrents.Update(t)
		return t
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if t.IsReady() {
				t.State = "pausedUP"
				s.torrents.Update(t)
				return t
			}

			updatedT := s.updateTorrent(t, debridTorrent)
			t = updatedT

		case <-time.After(10 * time.Minute): // Add a timeout
			return t
		}
	}
}
