package torbox

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"runtime"
	"time"

	"github.com/dylanmazurek/decypharr/internal/config"
	"github.com/dylanmazurek/decypharr/internal/logger"
	"github.com/dylanmazurek/decypharr/internal/request"
	"github.com/dylanmazurek/decypharr/internal/utils"
	debridModels "github.com/dylanmazurek/decypharr/pkg/debrid/models"
	"github.com/dylanmazurek/decypharr/pkg/debrid/providers/torbox/models"
	"github.com/dylanmazurek/decypharr/pkg/version"
	"github.com/rs/zerolog"
)

type Torbox struct {
	logger zerolog.Logger

	clientOptions *debridModels.ClientOptions

	client   *request.Client
	accounts *debridModels.Accounts
	profile  *debridModels.Profile
}

var _ debridModels.Client = (*Torbox)(nil)

func New(dc config.Debrid) (*Torbox, error) {
	rl := request.ParseRateLimit(dc.RateLimit)

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", dc.APIKey),
		"User-Agent":    fmt.Sprintf("Decypharr/%s (%s; %s)", version.GetInfo(), runtime.GOOS, runtime.GOARCH),
	}

	_log := logger.New(dc.Name)
	client := request.New(
		request.WithHeaders(headers),
		request.WithRateLimiter(rl),
		request.WithLogger(_log),
		request.WithProxy(dc.Proxy),
	)

	autoExpireLinksAfter, err := time.ParseDuration(dc.AutoExpireLinksAfter)
	if autoExpireLinksAfter == 0 || err != nil {
		autoExpireLinksAfter = 48 * time.Hour
	}

	newClientOptions := &debridModels.ClientOptions{
		Name:      "torbox",
		MountPath: dc.Folder,
		Host:      "https://api.torbox.app/v1",
		APIKey:    dc.APIKey,

		CheckCached:          dc.CheckCached,
		AddSamples:           dc.AddSamples,
		DownloadUncached:     dc.DownloadUncached,
		AutoExpireLinksAfter: autoExpireLinksAfter,
	}

	newTorbox := &Torbox{
		clientOptions: newClientOptions,
		client:        client,

		accounts: debridModels.NewAccounts(dc),
	}

	return newTorbox, nil
}

func (tb *Torbox) ClientOptions() debridModels.ClientOptions {
	return *tb.clientOptions
}

func (tb *Torbox) getTorboxStatus(status string, finished bool) string {
	if finished {
		return "downloaded"
	}

	cleanStatus := regexp.MustCompile(`\s*\(.*?\)\s*`).ReplaceAllString(status, "")

	downloadedStatuses := []string{
		"completed", "cached", "downloaded",
	}

	downloadingStatuses := []string{
		"paused", "downloading", "pausedDL",
		"pausedUP", "queuedUP", "forcedUP",
		"metaDL", "stopped seeding", "stalled",
		"queuedDL", "forcedDL", "moving", "allocating",
		"checkingUP", "checkingDL", "checkingResumeData",
	}

	seedingStatuses := []string{
		"seeding", "uploading",
	}

	switch {
	case utils.Contains(downloadedStatuses, cleanStatus):
		return "downloaded"
	case utils.Contains(downloadingStatuses, cleanStatus):
		return "downloading"
	case utils.Contains(seedingStatuses, cleanStatus):
		return "seeding"
	}

	return "error"
}

func (tb *Torbox) CheckStatus(torrent *debridModels.DebridTorrent) (*debridModels.DebridTorrent, error) {
	for {
		err := tb.UpdateTorrent(torrent)

		if err != nil || torrent == nil {
			return torrent, err
		}

		status := torrent.Status

		if status == "downloaded" {
			tb.logger.Info().Msgf("torrent: %s downloaded", torrent.Name)

			return torrent, nil
		} else if utils.Contains(tb.ClientOptions().DownloadingStatus, status) {
			if !torrent.DownloadUncached {
				return torrent, fmt.Errorf("torrent: %s not cached", torrent.Name)
			}

			return torrent, nil
		} else {
			return torrent, fmt.Errorf("torrent: %s has error", torrent.Name)
		}
	}
}

func (tb *Torbox) GetAvailableSlots() (*int, error) {
	var planSlots map[string]int = map[string]int{
		"essential": 3,
		"standard":  5,
		"pro":       10,
	}

	var accountSlots int = 1
	profile, err := tb.GetProfile()
	if err != nil {
		return nil, err
	}

	if slots, ok := planSlots[profile.Type]; ok {
		accountSlots = slots
	}

	activeTorrents, err := tb.GetTorrents()
	if err != nil {
		return nil, err
	}

	activeCount := 0
	for _, t := range activeTorrents {
		if t.IsActive != nil && !*t.IsActive {
			activeCount++
		}
	}

	available := max(accountSlots-activeCount, 0)

	return &available, nil
}

func (tb *Torbox) GetProfile() (*debridModels.Profile, error) {
	if tb.profile != nil {
		return tb.profile, nil
	}

	url := fmt.Sprintf("%s/api/user/me?settings=true", tb.ClientOptions().Host)
	req, _ := http.NewRequest(http.MethodGet, url, nil)

	resp, err := tb.client.MakeRequest(req)
	if err != nil {
		return nil, err
	}

	var profileResp models.GetProfileResponse
	err = json.Unmarshal(resp, &profileResp)
	if err != nil {
		return nil, err
	}

	if !profileResp.Success || profileResp.Data == nil {
		return nil, fmt.Errorf("error getting profile: %v", profileResp.Error)
	}

	userData := profileResp.Data

	profile := &debridModels.Profile{
		Name:       tb.ClientOptions().Name,
		Id:         userData.Id,
		Username:   userData.Email,
		Email:      userData.Email,
		Expiration: userData.PremiumExpiresAt,
	}

	switch userData.Plan {
	case 1:
		profile.Type = "essential"
	case 2:
		profile.Type = "pro"
	case 3:
		profile.Type = "standard"
	default:
		profile.Type = "free"
	}

	tb.profile = profile

	return profile, nil
}

func (tb *Torbox) GetAccounts() *debridModels.Accounts {
	return tb.accounts
}

func (tb *Torbox) SyncAccounts() error {
	return nil
}
