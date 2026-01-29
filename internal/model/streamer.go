package model

import "time"

type Platform string

const (
	PlatformBilibili Platform = "bilibili"
	PlatformDouyin   Platform = "douyin"
	PlatformKuaishou Platform = "kuaishou"
	PlatformDouyu    Platform = "douyu"
	PlatformCC163    Platform = "cc163"
	PlatformWeibo    Platform = "weibo"
)

type Streamer struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Platform        Platform `json:"platform"`
	RoomID          string   `json:"room_id"`
	Avatar          string   `json:"avatar"`
	AvatarLocal     string   `json:"avatar_local,omitempty"`
	IsLive          bool     `json:"is_live"`
	Title           string   `json:"title"`
	StartTime       string   `json:"start_time,omitempty"`
	ViewerCount     int64    `json:"viewer_count,omitempty"`
	RoomURL         string   `json:"room_url"`
	LastQueryTime   string   `json:"last_query_time,omitempty"`
	LastQueryFailed bool     `json:"last_query_failed,omitempty"`
}

type StreamerConfig struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Platform Platform `json:"platform"`
	RoomID   string   `json:"room_id"`
	Avatar   string   `json:"avatar,omitempty"`
	LiveURL  string   `json:"live_url,omitempty"`
}

type Config struct {
	Streamers []StreamerConfig `json:"streamers"`
}

type Settings struct {
	ScanIntervalMinutes     int `json:"scan_interval_minutes"`
	PlatformDelayMinSeconds int `json:"platform_delay_min_seconds"`
	PlatformDelayMaxSeconds int `json:"platform_delay_max_seconds"`
}

type LiveSession struct {
	ID         int64     `json:"id"`
	StreamerID string    `json:"streamer_id"`
	Platform   Platform  `json:"platform"`
	RoomID     string    `json:"room_id"`
	Title      string    `json:"title"`
	StartTime  time.Time `json:"start_time"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	Duration   int64     `json:"duration,omitempty"` // seconds
}

type StreamerStats struct {
	StreamerID     string  `json:"streamer_id"`
	TotalSessions  int     `json:"total_sessions"`
	TotalDuration  int64   `json:"total_duration"`  // seconds
	AvgDuration    int64   `json:"avg_duration"`    // seconds
	LastLiveTime   string  `json:"last_live_time,omitempty"`
	WeekSessions   int     `json:"week_sessions"`
	MonthSessions  int     `json:"month_sessions"`
}
