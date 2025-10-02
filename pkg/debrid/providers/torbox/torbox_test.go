package torbox

import (
	"os"
	"sync"
	"testing"

	"github.com/dylanmazurek/decypharr/internal/config"
	"github.com/dylanmazurek/decypharr/internal/utils"
	"github.com/dylanmazurek/decypharr/pkg/debrid/types"
)

var (
	testClient *Torbox
	clientOnce sync.Once
)

func getTestClient(t *testing.T) *Torbox {
	t.Helper()

	apiKey := os.Getenv("TORBOX_API_KEY")
	if apiKey == "" {
		t.Skip("TORBOX_API_KEY not set")
	}

	clientOnce.Do(func() {
		config.SetConfigPath("env")
		c, err := New(config.Debrid{
			Name:   "torbox",
			APIKey: apiKey,
			Folder: "/decypharr",
		})
		if err != nil {
			t.Fatalf("failed to create Torbox client: %v", err)
		}

		testClient = c
	})

	if testClient == nil {
		t.Fatal("Torbox client was not initialized")
	}

	return testClient
}

var (
	torrentId string = "4516057"
)

func TestGetProfile(t *testing.T) {
	testClient := getTestClient(t)

	profile, err := testClient.GetProfile()
	if err != nil {
		t.Fatalf("failed to get profile: %v", err)
	}

	if profile.Username == "" {
		t.Fatal("expected username to be set")
	}
}

func TestAddMagnet(t *testing.T) {
	testClient := getTestClient(t)

	newMagnet, err := utils.GetMagnetFromUrl("magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel&xs=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2Fsintel.torrent")
	if err != nil {
		t.Fatalf("failed to parse magnet: %v", err)
	}

	newTorrent := &types.Torrent{
		Magnet:   newMagnet,
		InfoHash: newMagnet.InfoHash,
		Name:     newMagnet.Name,
		Size:     newMagnet.Size,
		Files:    make(map[string]types.File),
	}

	torrent, err := testClient.SubmitMagnet(newTorrent)
	if err != nil {
		t.Fatalf("failed to add magnet: %v", err)
	}

	if torrent.Id == "" {
		t.Fatal("expected torrent ID to be set")
	}

	torrentId = torrent.Id
}

func TestGetTorrent(t *testing.T) {
	testClient := getTestClient(t)

	torrent, err := testClient.GetTorrent(torrentId)
	if err != nil {
		t.Fatalf("failed to get torrent: %v", err)
	}

	if torrent.Id != torrentId {
		t.Fatalf("expected torrent ID %s, got %s", torrentId, torrent.Id)
	}
}

func TestUpdateTorrent(t *testing.T) {
	testClient := getTestClient(t)

	updateTorrent := &types.Torrent{
		Id:   torrentId,
		Size: 22,
	}

	err := testClient.UpdateTorrent(updateTorrent)
	if err != nil {
		t.Fatalf("failed to update torrent: %v", err)
	}
}

func TestDeleteTorrent(t *testing.T) {
	testClient := getTestClient(t)

	err := testClient.DeleteTorrent(torrentId)
	if err != nil {
		t.Fatalf("failed to delete torrent: %v", err)
	}

	torrentId = ""
}

func TestGetAvailableSlots(t *testing.T) {
	testClient := getTestClient(t)
	slots, err := testClient.GetAvailableSlots()
	if err != nil {
		t.Fatalf("failed to get available slots: %v", err)
	}

	if slots < 0 {
		t.Fatalf("expected available slots to be non-negative, got %d", slots)
	}

	t.Logf("Available slots: %d", slots)
}
