package crawler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cxtv-alerts/internal/model"
)

type DouyuCrawler struct {
	client *http.Client
}

func NewDouyuCrawler() *DouyuCrawler {
	return &DouyuCrawler{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *DouyuCrawler) Platform() model.Platform {
	return model.PlatformDouyu
}

type douyuAvatar struct {
	Big    string `json:"big"`
	Middle string `json:"middle"`
	Small  string `json:"small"`
}

type douyuResponse struct {
	Error int `json:"error"`
	Room  struct {
		RoomID     int         `json:"room_id"`
		RoomName   string      `json:"room_name"`
		Nickname   string      `json:"nickname"`
		Avatar     douyuAvatar `json:"avatar"`
		AvatarMid  string      `json:"avatar_mid"`
		ShowStatus int         `json:"show_status"` // 1 = live, 2 = offline
		Online     int64       `json:"online"`
		ShowTime   int64       `json:"show_time"`
	} `json:"room"`
}

func (c *DouyuCrawler) GetLiveStatus(roomID string) (*model.Streamer, error) {
	url := fmt.Sprintf("https://www.douyu.com/betard/%s", roomID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result douyuResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != 0 {
		return nil, fmt.Errorf("douyu API error: %d", result.Error)
	}

	avatar := result.Room.AvatarMid
	if avatar == "" {
		avatar = result.Room.Avatar.Middle
	}

	var startTime string
	if result.Room.ShowTime > 0 {
		t := time.Unix(result.Room.ShowTime, 0)
		startTime = t.Format("2006-01-02 15:04:05")
	}

	streamer := &model.Streamer{
		Platform:    model.PlatformDouyu,
		RoomID:      roomID,
		Name:        result.Room.Nickname,
		Title:       result.Room.RoomName,
		Avatar:      avatar,
		IsLive:      result.Room.ShowStatus == 1,
		ViewerCount: result.Room.Online,
		StartTime:   startTime,
		RoomURL:     fmt.Sprintf("https://www.douyu.com/%s", roomID),
	}

	return streamer, nil
}
