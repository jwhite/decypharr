package torbox

import (
	"encoding/json"
	"fmt"
	"net/http"
	gourl "net/url"
	"sync"
	"time"

	debridModels "github.com/dylanmazurek/decypharr/pkg/debrid/models"
	"github.com/dylanmazurek/decypharr/pkg/providers/torbox/models"
)

func (tb *Torbox) GetDownloadLink(t *debridModels.DebridTorrent, file *debridModels.File) (*debridModels.DownloadLink, error) {
	url := fmt.Sprintf("%s/api/torrents/requestdl/", tb.Host)

	query := gourl.Values{}
	query.Add("torrent_id", t.Id)
	query.Add("token", tb.APIKey)
	query.Add("file_id", file.Id)
	url += "?" + query.Encode()

	req, _ := http.NewRequest(http.MethodGet, url, nil)
	resp, err := tb.client.MakeRequest(req)
	if err != nil {
		tb.logger.Error().
			Err(err).
			Str("torrent_id", t.Id).
			Str("file_id", file.Id).
			Msg("Failed to make request to Torbox API")

		return nil, err
	}

	var data models.DownloadLinksResponse
	if err = json.Unmarshal(resp, &data); err != nil {
		tb.logger.Error().
			Err(err).
			Str("torrent_id", t.Id).
			Str("file_id", file.Id).
			Msg("Failed to unmarshal Torbox API response")

		return nil, err
	}

	if data.Data == nil {
		tb.logger.Error().
			Str("torrent_id", t.Id).
			Str("file_id", file.Id).
			Bool("success", data.Success).
			Interface("error", data.Error).
			Str("detail", data.Detail).
			Msg("Torbox API returned no data")

		return nil, fmt.Errorf("error getting download links")
	}

	link := *data.Data
	if link == "" {
		tb.logger.Error().
			Str("torrent_id", t.Id).
			Str("file_id", file.Id).
			Msg("Torbox API returned empty download link")

		return nil, fmt.Errorf("error getting download links")
	}

	now := time.Now()
	downloadLink := &models.DownloadLink{
		Link:         file.Link,
		DownloadLink: link,
		Id:           file.Id,
		Generated:    now,
		ExpiresAt:    now.Add(tb.autoExpiresLinksAfter),
	}

	return downloadLink, nil
}

func (tb *Torbox) GetDownloadLinks() (map[string]*debridModels.DownloadLink, error) {
	return nil, nil
}

func (tb *Torbox) CheckDownloadLink(link string) error {
	return nil
}

func (tb *Torbox) DeleteDownloadLink(linkId string) error {
	return nil
}

func (tb *Torbox) GetFileDownloadLinks(t *debridModels.DebridTorrent) error {
	filesCh := make(chan debridModels.File, len(t.Files))
	linkCh := make(chan *debridModels.DownloadLink)
	errCh := make(chan error, len(t.Files))

	var wg sync.WaitGroup
	wg.Add(len(t.Files))
	for _, file := range t.Files {
		go func() {
			defer wg.Done()
			link, err := tb.GetDownloadLink(t, &file)
			if err != nil {
				errCh <- err
				return
			}
			if link != nil {
				linkCh <- link
				file.DownloadLink = link
			}
			filesCh <- file
		}()
	}
	go func() {
		wg.Wait()
		close(filesCh)
		close(linkCh)
		close(errCh)
	}()

	// Collect download links before files to ensure all download operations are completed
	// and available before updating the files map. This order prevents potential race conditions
	// and ensures proper completion of download operations. See issue #123 for details.
	for link := range linkCh {
		if link != nil {
			tb.accounts.SetDownloadLink(link.Link, link)
		}
	}

	files := make(map[string]debridModels.File, len(t.Files))
	for file := range filesCh {
		files[file.Name] = file
	}

	// Check for errors
	for err := range errCh {
		if err != nil {
			return err // Return the first error encountered
		}
	}

	t.Files = files
	return nil
}
