package service

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const avatarDir = "web/avatars"

func (s *Service) StartAvatarUpdater() {
	// Update avatars immediately on startup
	go s.updateAllAvatars()

	// Then update daily
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			s.updateAllAvatars()
		}
	}()
}

func (s *Service) updateAllAvatars() {
	log.Println("Starting avatar update...")

	for _, sc := range s.config.Streamers {
		avatarURL := sc.Avatar
		if avatarURL == "" {
			continue
		}

		// Check if we need to update
		_, currentLocal, lastUpdated, err := s.db.GetAvatarInfo(sc.ID)
		if err != nil {
			log.Printf("Error getting avatar info for %s: %v", sc.Name, err)
			continue
		}

		// Skip if updated within 24 hours and local file exists
		if lastUpdated != nil && time.Since(*lastUpdated) < 24*time.Hour {
			if currentLocal != "" {
				localPath := filepath.Join(avatarDir, currentLocal)
				if _, err := os.Stat(localPath); err == nil {
					continue
				}
			}
		}

		// Download avatar
		localFile, err := s.downloadAvatar(sc.ID, avatarURL)
		if err != nil {
			log.Printf("Error downloading avatar for %s: %v", sc.Name, err)
			continue
		}

		// Update database
		if err := s.db.UpdateAvatar(sc.ID, avatarURL, localFile); err != nil {
			log.Printf("Error updating avatar record for %s: %v", sc.Name, err)
			continue
		}

		// Update in-memory streamer
		s.mu.Lock()
		if streamer, ok := s.streamers[sc.ID]; ok {
			streamer.AvatarLocal = "/static/avatars/" + localFile
		}
		s.mu.Unlock()

		// Small delay between downloads
		time.Sleep(100 * time.Millisecond)
	}

	log.Println("Avatar update complete")
}

func (s *Service) downloadAvatar(streamerID, url string) (string, error) {
	// Create a unique filename based on streamer ID and URL hash
	hash := fmt.Sprintf("%x", md5.Sum([]byte(url)))[:8]
	ext := getExtension(url)
	filename := fmt.Sprintf("%s_%s%s", streamerID, hash, ext)
	localPath := filepath.Join(avatarDir, filename)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Create output file
	out, err := os.Create(localPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Copy response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(localPath)
		return "", err
	}

	return filename, nil
}

func getExtension(url string) string {
	// Remove query string
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}

	ext := filepath.Ext(url)
	if ext == "" || len(ext) > 5 {
		return ".jpg" // Default extension
	}

	// Normalize extension
	ext = strings.ToLower(ext)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return ext
	default:
		return ".jpg"
	}
}

func (s *Service) loadLocalAvatars() {
	for id, streamer := range s.streamers {
		_, avatarLocal, _, err := s.db.GetAvatarInfo(id)
		if err != nil {
			continue
		}
		if avatarLocal != "" {
			localPath := filepath.Join(avatarDir, avatarLocal)
			if _, err := os.Stat(localPath); err == nil {
				streamer.AvatarLocal = "/static/avatars/" + avatarLocal
			}
		}
	}
}
