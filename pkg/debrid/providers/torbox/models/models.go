package models

type BaseResponse[T any] struct {
	Success bool   `json:"success"`
	Error   any    `json:"error"`
	Detail  string `json:"detail"`

	Data *T `json:"data"`
}

type BasicTorrentResponse struct {
	ID     *int    `json:"id,omitempty"`
	Name   *string `json:"name,omitempty"`
	Size   *int64  `json:"size,omitempty"`
	Hash   *string `json:"hash,omitempty"`
	Status *string `json:"status,omitempty"`
}

type AvailableResponse BaseResponse[map[string]BasicTorrentResponse]

type CreateTorrentResponse BaseResponse[BasicTorrentResponse]

type GetProfileResponse BaseResponse[TorboxProfile]

type InfoResponse BaseResponse[Torrent]

type DownloadLinksResponse BaseResponse[string]

type TorrentsListResponse BaseResponse[[]Torrent]
