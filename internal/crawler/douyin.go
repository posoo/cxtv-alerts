package crawler

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"cxtv-alerts/internal/model"
)

type DouyinCrawler struct {
	client *http.Client
}

func NewDouyinCrawler() *DouyinCrawler {
	return &DouyinCrawler{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *DouyinCrawler) Platform() model.Platform {
	return model.PlatformDouyin
}

func (c *DouyinCrawler) GetLiveStatus(roomID string) (*model.Streamer, error) {
	url := fmt.Sprintf("https://live.douyin.com/%s", roomID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Cookie", "__ac_nonce=0123456789")

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
		Platform: model.PlatformDouyin,
		RoomID:   roomID,
		RoomURL:  url,
		IsLive:   false,
	}

	// Check if live - look for live indicators in the page
	if strings.Contains(html, `"status":2`) || strings.Contains(html, `"alive":true`) {
		streamer.IsLive = true
	}

	// Extract title
	titleRe := regexp.MustCompile(`<title>([^<]+)</title>`)
	if matches := titleRe.FindStringSubmatch(html); len(matches) > 1 {
		title := matches[1]
		title = strings.TrimSuffix(title, " - 抖音直播")
		title = strings.TrimSuffix(title, "_抖音直播")
		streamer.Title = title
	}

	// Extract nickname from meta or script
	nicknameRe := regexp.MustCompile(`"nickname":"([^"]+)"`)
	if matches := nicknameRe.FindStringSubmatch(html); len(matches) > 1 {
		streamer.Name = matches[1]
	}

	// Extract avatar
	avatarRe := regexp.MustCompile(`"avatar_thumb":\{"url_list":\["([^"]+)"`)
	if matches := avatarRe.FindStringSubmatch(html); len(matches) > 1 {
		streamer.Avatar = strings.ReplaceAll(matches[1], "\\u0026", "&")
	}

	return streamer, nil
}
