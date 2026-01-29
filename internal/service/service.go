package service

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"cxtv-alerts/internal/crawler"
	"cxtv-alerts/internal/database"
	"cxtv-alerts/internal/model"
)

type Service struct {
	db          *database.DB
	crawlers    map[model.Platform]crawler.Crawler
	config      *model.Config
	settings    *model.Settings
	streamers   map[string]*model.Streamer
	sessions    map[string]int64 // streamerID -> sessionID
	errorCounts map[string]int   // streamerID -> consecutive error count
	mu          sync.RWMutex
}

func New(db *database.DB, configPath, settingsPath string) (*Service, error) {
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	settings, err := loadSettings(settingsPath)
	if err != nil {
		log.Printf("Warning: failed to load settings, using defaults: %v", err)
		settings = &model.Settings{
			ScanIntervalMinutes:     5,
			PlatformDelayMinSeconds: 5,
			PlatformDelayMaxSeconds: 20,
		}
	}

	s := &Service{
		db:          db,
		config:      config,
		settings:    settings,
		streamers:   make(map[string]*model.Streamer),
		sessions:    make(map[string]int64),
		errorCounts: make(map[string]int),
		crawlers: map[model.Platform]crawler.Crawler{
			model.PlatformBilibili: crawler.NewBilibiliCrawler(),
			model.PlatformDouyu:    crawler.NewDouyuCrawler(),
			model.PlatformDouyin:   crawler.NewDouyinCrawler(),
			model.PlatformKuaishou: crawler.NewKuaishouCrawler(),
			model.PlatformCC163:    crawler.NewCC163Crawler(),
			model.PlatformWeibo:    crawler.NewWeiboCrawler(),
		},
	}

	// Initialize streamers from config
	for _, sc := range config.Streamers {
		s.streamers[sc.ID] = &model.Streamer{
			ID:       sc.ID,
			Name:     sc.Name,
			Platform: sc.Platform,
			RoomID:   sc.RoomID,
			Avatar:   sc.Avatar,
			RoomURL:  sc.LiveURL,
			IsLive:   false,
		}
	}

	// Restore status from database
	for id := range s.streamers {
		// Restore active sessions
		if session, err := db.GetActiveSession(id); err == nil && session != nil {
			s.sessions[id] = session.ID
			s.streamers[id].IsLive = true
			s.streamers[id].StartTime = session.StartTime.Format("2006-01-02 15:04:05")
		}

		// Restore last query time and cached status
		if lastTime, lastFailed, isLive, title, viewerCount, avatarLocal, err := db.GetStreamerStatus(id); err == nil {
			if lastTime != nil {
				s.streamers[id].LastQueryTime = lastTime.Format("2006-01-02 15:04:05")
			}
			s.streamers[id].LastQueryFailed = lastFailed
			if avatarLocal != "" {
				s.streamers[id].AvatarLocal = "/static/avatars/" + avatarLocal
			}
			// Only use cached status if we don't have active session info
			if _, hasSession := s.sessions[id]; !hasSession {
				s.streamers[id].IsLive = isLive
				s.streamers[id].Title = title
				s.streamers[id].ViewerCount = viewerCount
			}
		}
	}

	// Load local avatars
	s.loadLocalAvatars()

	return s, nil
}

func loadConfig(path string) (*model.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config model.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func loadSettings(path string) (*model.Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var settings model.Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

func (s *Service) StartScanner() {
	interval := time.Duration(s.settings.ScanIntervalMinutes) * time.Minute
	log.Printf("Scanner started with interval: %v", interval)

	// Do an immediate scan
	go s.scan()

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			s.scan()
		}
	}()
}

func (s *Service) scan() {
	log.Println("Starting scan...")

	// Group streamers by platform
	platformStreamers := make(map[model.Platform][]model.StreamerConfig)
	for _, sc := range s.config.Streamers {
		platformStreamers[sc.Platform] = append(platformStreamers[sc.Platform], sc)
	}

	var wg sync.WaitGroup

	// Scan each platform in parallel, but within each platform scan sequentially with random delay
	for platform, streamers := range platformStreamers {
		c, ok := s.crawlers[platform]
		if !ok {
			log.Printf("No crawler for platform: %s", platform)
			continue
		}

		wg.Add(1)
		go func(p model.Platform, scs []model.StreamerConfig, cr crawler.Crawler) {
			defer wg.Done()
			s.scanPlatform(p, scs, cr)
		}(platform, streamers, c)
	}

	wg.Wait()
	log.Println("Scan complete")
}

func (s *Service) scanPlatform(platform model.Platform, streamers []model.StreamerConfig, c crawler.Crawler) {
	scanInterval := time.Duration(s.settings.ScanIntervalMinutes) * time.Minute
	minDelay := s.settings.PlatformDelayMinSeconds
	maxDelay := s.settings.PlatformDelayMaxSeconds

	for i, sc := range streamers {
		// Check if we should skip this streamer based on last query time
		lastQueryTime, err := s.db.GetLastQueryTime(sc.ID)
		if err != nil {
			log.Printf("Error getting last query time for %s: %v", sc.Name, err)
		} else if lastQueryTime != nil {
			elapsed := time.Since(*lastQueryTime)
			if elapsed < scanInterval {
				// Skip this streamer, was queried recently
				continue
			}
		}

		// Scan the streamer
		s.scanStreamer(sc, c)

		// Add random delay between requests (except for last one)
		if i < len(streamers)-1 {
			delay := time.Duration(minDelay+rand.Intn(maxDelay-minDelay+1)) * time.Second
			time.Sleep(delay)
		}
	}
}

func (s *Service) scanStreamer(sc model.StreamerConfig, c crawler.Crawler) {
	result, err := c.GetLiveStatus(sc.RoomID)
	now := time.Now().Format("2006-01-02 15:04:05")

	if err != nil {
		s.mu.Lock()
		s.errorCounts[sc.ID]++
		count := s.errorCounts[sc.ID]
		// Mark as failed
		streamer := s.streamers[sc.ID]
		streamer.LastQueryTime = now
		streamer.LastQueryFailed = true
		s.mu.Unlock()

		// Update database with failed status
		s.db.UpdateStreamerStatus(sc.ID, false, "", 0, true)

		// Only log error on first occurrence or every 10th consecutive error
		if count == 1 || count%10 == 0 {
			log.Printf("Error scanning %s (%s): %v (count: %d)", sc.Name, sc.Platform, err, count)
		}
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Reset error count on success
	s.errorCounts[sc.ID] = 0

	streamer := s.streamers[sc.ID]
	wasLive := streamer.IsLive

	// Update streamer info
	streamer.IsLive = result.IsLive
	streamer.Title = result.Title
	streamer.ViewerCount = result.ViewerCount
	streamer.LastQueryTime = now
	streamer.LastQueryFailed = false

	// Keep RoomURL from config, only update if crawler provides one and config doesn't have it
	if streamer.RoomURL == "" && result.RoomURL != "" {
		streamer.RoomURL = result.RoomURL
	}
	if result.Avatar != "" {
		streamer.Avatar = result.Avatar
	}
	if result.Name != "" {
		streamer.Name = result.Name
	}
	if result.StartTime != "" {
		streamer.StartTime = result.StartTime
	}

	// Update database with query time and status
	if err := s.db.UpdateStreamerStatus(sc.ID, result.IsLive, result.Title, result.ViewerCount, false); err != nil {
		log.Printf("Error updating streamer status for %s: %v", sc.Name, err)
	}

	// Handle session tracking
	if result.IsLive && !wasLive {
		// Started streaming
		sessionID, err := s.db.StartSession(sc.ID, sc.Platform, sc.RoomID, result.Title)
		if err != nil {
			log.Printf("Error starting session for %s: %v", sc.Name, err)
		} else {
			s.sessions[sc.ID] = sessionID
			log.Printf("%s started streaming: %s", sc.Name, result.Title)
		}
	} else if !result.IsLive && wasLive {
		// Stopped streaming
		if sessionID, ok := s.sessions[sc.ID]; ok {
			if err := s.db.EndSession(sessionID); err != nil {
				log.Printf("Error ending session for %s: %v", sc.Name, err)
			} else {
				delete(s.sessions, sc.ID)
				log.Printf("%s stopped streaming", sc.Name)
			}
		}
		streamer.StartTime = ""
	}
}

func (s *Service) GetStreamers() []*model.Streamer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*model.Streamer, 0, len(s.streamers))
	for _, streamer := range s.streamers {
		// Create a copy to avoid race conditions
		copy := *streamer
		result = append(result, &copy)
	}

	return result
}

func (s *Service) GetHistory(streamerID string, limit int) ([]model.LiveSession, error) {
	return s.db.GetHistory(streamerID, limit)
}

func (s *Service) GetStats(streamerID string) (*model.StreamerStats, error) {
	return s.db.GetStats(streamerID)
}
