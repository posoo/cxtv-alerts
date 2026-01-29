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

type CC163Crawler struct {
	client *http.Client
}

func NewCC163Crawler() *CC163Crawler {
	return &CC163Crawler{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *CC163Crawler) Platform() model.Platform {
	return model.PlatformCC163
}

func (c *CC163Crawler) GetLiveStatus(roomID string) (*model.Streamer, error) {
	url := fmt.Sprintf("https://cc.163.com/%s/", roomID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

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
		Platform: model.PlatformCC163,
		RoomID:   roomID,
		RoomURL:  url,
		IsLive:   false,
	}

	// Look for room data in script
	dataRe := regexp.MustCompile(`var\s+roomData\s*=\s*(\{[^;]+\});`)
	if matches := dataRe.FindStringSubmatch(html); len(matches) > 1 {
		var data map[string]interface{}
		if json.Unmarshal([]byte(matches[1]), &data) == nil {
			if status, ok := data["isLive"].(float64); ok {
				streamer.IsLive = status == 1
			}
			if title, ok := data["title"].(string); ok {
				streamer.Title = title
			}
			if nickname, ok := data["nickname"].(string); ok {
				streamer.Name = nickname
			}
			if avatar, ok := data["purl"].(string); ok {
				streamer.Avatar = avatar
			}
		}
	}

	// Alternative: look for __NEXT_DATA__ or similar
	nextDataRe := regexp.MustCompile(`<script id="__NEXT_DATA__"[^>]*>(\{.+?\})</script>`)
	if matches := nextDataRe.FindStringSubmatch(html); len(matches) > 1 {
		var data map[string]interface{}
		if json.Unmarshal([]byte(matches[1]), &data) == nil {
			// Navigate through the data structure
			if props, ok := data["props"].(map[string]interface{}); ok {
				if pageProps, ok := props["pageProps"].(map[string]interface{}); ok {
					if roomInfo, ok := pageProps["roomInfo"].(map[string]interface{}); ok {
						if isLive, ok := roomInfo["isLive"].(bool); ok {
							streamer.IsLive = isLive
						}
						if title, ok := roomInfo["title"].(string); ok {
							streamer.Title = title
						}
						if nickname, ok := roomInfo["nickname"].(string); ok {
							streamer.Name = nickname
						}
					}
				}
			}
		}
	}

	// Fallback title extraction
	if streamer.Title == "" {
		titleRe := regexp.MustCompile(`<title>([^<]+)</title>`)
		if matches := titleRe.FindStringSubmatch(html); len(matches) > 1 {
			streamer.Title = matches[1]
		}
	}

	return streamer, nil
}
