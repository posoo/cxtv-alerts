package crawler

import "cxtv-alerts/internal/model"

type Crawler interface {
	GetLiveStatus(roomID string) (*model.Streamer, error)
	Platform() model.Platform
}
