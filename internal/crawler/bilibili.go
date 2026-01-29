package crawler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cxtv-alerts/internal/model"
)

type BilibiliCrawler struct {
	client *http.Client
}

func NewBilibiliCrawler() *BilibiliCrawler {
	return &BilibiliCrawler{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *BilibiliCrawler) Platform() model.Platform {
	return model.PlatformBilibili
}

type bilibiliResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		RoomID        int64  `json:"room_id"`
		ShortID       int    `json:"short_id"`
		UID           int64  `json:"uid"`
		Title         string `json:"title"`
		LiveStatus    int    `json:"live_status"`
		LiveStartTime int64  `json:"live_start_time"`
		Online        int64  `json:"online"`
	} `json:"data"`
}

type bilibiliUserResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Info struct {
			UID   int64  `json:"uid"`
			UName string `json:"uname"`
			Face  string `json:"face"`
		} `json:"info"`
	} `json:"data"`
}

func (c *BilibiliCrawler) GetLiveStatus(roomID string) (*model.Streamer, error) {
	url := fmt.Sprintf("https://api.live.bilibili.com/room/v1/Room/get_info?room_id=%s", roomID)

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

	var result bilibiliResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("bilibili API error: %s", result.Message)
	}

	streamer := &model.Streamer{
		Platform:    model.PlatformBilibili,
		RoomID:      roomID,
		Title:       result.Data.Title,
		IsLive:      result.Data.LiveStatus == 1,
		ViewerCount: result.Data.Online,
		RoomURL:     fmt.Sprintf("https://live.bilibili.com/%s", roomID),
	}

	if result.Data.LiveStartTime > 0 {
		t := time.Unix(result.Data.LiveStartTime, 0)
		streamer.StartTime = t.Format("2006-01-02 15:04:05")
	}

	// Get user info for avatar
	userURL := fmt.Sprintf("https://api.live.bilibili.com/live_user/v1/UserInfo/get_anchor_in_room?roomid=%s", roomID)
	userReq, err := http.NewRequest("GET", userURL, nil)
	if err == nil {
		userReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		userResp, err := c.client.Do(userReq)
		if err == nil {
			defer userResp.Body.Close()
			var userResult bilibiliUserResponse
			if json.NewDecoder(userResp.Body).Decode(&userResult) == nil && userResult.Code == 0 {
				streamer.Avatar = userResult.Data.Info.Face
				if streamer.Name == "" {
					streamer.Name = userResult.Data.Info.UName
				}
			}
		}
	}

	return streamer, nil
}
