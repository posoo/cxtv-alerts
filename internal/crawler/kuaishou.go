package crawler

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"cxtv-alerts/internal/model"
)

type KuaishouCrawler struct {
	client *http.Client
}

func NewKuaishouCrawler() *KuaishouCrawler {
	// Custom transport with TLS config to handle Kuaishou's requirements
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		DisableKeepAlives: true,
	}

	return &KuaishouCrawler{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

func (c *KuaishouCrawler) Platform() model.Platform {
	return model.PlatformKuaishou
}

func (c *KuaishouCrawler) GetLiveStatus(roomID string) (*model.Streamer, error) {
	url := fmt.Sprintf("https://live.kuaishou.com/u/%s", roomID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// More complete browser headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Sec-Ch-Ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

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
		Platform: model.PlatformKuaishou,
		RoomID:   roomID,
		RoomURL:  url,
		IsLive:   false,
	}

	// Look for __INITIAL_STATE__ or similar data
	stateRe := regexp.MustCompile(`__INITIAL_STATE__\s*=\s*(\{.+?\});`)
	if matches := stateRe.FindStringSubmatch(html); len(matches) > 1 {
		var data map[string]interface{}
		if json.Unmarshal([]byte(matches[1]), &data) == nil {
			if liveStream, ok := data["liveStream"].(map[string]interface{}); ok {
				if living, ok := liveStream["living"].(bool); ok {
					streamer.IsLive = living
				}
				if caption, ok := liveStream["caption"].(string); ok {
					streamer.Title = caption
				}
			}
			if author, ok := data["author"].(map[string]interface{}); ok {
				if name, ok := author["name"].(string); ok {
					streamer.Name = name
				}
				if avatar, ok := author["avatar"].(string); ok {
					streamer.Avatar = avatar
				}
			}
		}
	}

	// Fallback: check for live indicators in different formats
	if strings.Contains(html, `"living":true`) || strings.Contains(html, `"isLiving":true`) {
		streamer.IsLive = true
	}

	// Extract title from page title
	if streamer.Title == "" {
		titleRe := regexp.MustCompile(`<title>([^<]+)</title>`)
		if matches := titleRe.FindStringSubmatch(html); len(matches) > 1 {
			title := matches[1]
			title = strings.TrimSuffix(title, " - 快手直播")
			title = strings.TrimSuffix(title, " - 快手")
			streamer.Title = title
		}
	}

	return streamer, nil
}
