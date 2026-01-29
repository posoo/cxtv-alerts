package crawler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"cxtv-alerts/internal/model"
)

type WeiboCrawler struct {
	client *http.Client
}

func NewWeiboCrawler() *WeiboCrawler {
	return &WeiboCrawler{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *WeiboCrawler) Platform() model.Platform {
	return model.PlatformWeibo
}

func (c *WeiboCrawler) GetLiveStatus(roomID string) (*model.Streamer, error) {
	// Weibo live room URL
	url := fmt.Sprintf("https://weibo.com/l/wblive/p/show/%s", roomID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	html := string(body)

	streamer := &model.Streamer{
		Platform: model.PlatformWeibo,
		RoomID:   roomID,
		RoomURL:  url,
		IsLive:   false,
	}

	// Look for render data
	dataRe := regexp.MustCompile(`<script>window\.__INITIAL_STATE__\s*=\s*(\{.+?\});</script>`)
	if matches := dataRe.FindStringSubmatch(html); len(matches) > 1 {
		var data map[string]interface{}
		if json.Unmarshal([]byte(matches[1]), &data) == nil {
			if liveInfo, ok := data["liveInfo"].(map[string]interface{}); ok {
				if status, ok := liveInfo["status"].(float64); ok {
					streamer.IsLive = status == 1
				}
				if title, ok := liveInfo["title"].(string); ok {
					streamer.Title = title
				}
			}
			if userInfo, ok := data["userInfo"].(map[string]interface{}); ok {
				if name, ok := userInfo["screen_name"].(string); ok {
					streamer.Name = name
				}
				if avatar, ok := userInfo["avatar_large"].(string); ok {
					streamer.Avatar = avatar
				}
			}
		}
	}

	// Alternative: check for live status indicator
	if regexp.MustCompile(`"status":\s*1`).MatchString(html) || regexp.MustCompile(`"living":\s*true`).MatchString(html) {
		streamer.IsLive = true
	}

	// Extract title from page
	if streamer.Title == "" {
		titleRe := regexp.MustCompile(`<title>([^<]+)</title>`)
		if matches := titleRe.FindStringSubmatch(html); len(matches) > 1 {
			streamer.Title = matches[1]
		}
	}

	return streamer, nil
}
