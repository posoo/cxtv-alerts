package database

import (
	"database/sql"
	"time"

	"cxtv-alerts/internal/model"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) migrate() error {
	// Create tables
	query := `
	CREATE TABLE IF NOT EXISTS live_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		streamer_id TEXT NOT NULL,
		platform TEXT NOT NULL,
		room_id TEXT NOT NULL,
		title TEXT,
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_live_sessions_streamer ON live_sessions(streamer_id);
	CREATE INDEX IF NOT EXISTS idx_live_sessions_start_time ON live_sessions(start_time);

	CREATE TABLE IF NOT EXISTS streamer_status (
		streamer_id TEXT PRIMARY KEY,
		last_query_time DATETIME,
		is_live INTEGER DEFAULT 0,
		title TEXT,
		viewer_count INTEGER DEFAULT 0
	);
	`
	if _, err := db.conn.Exec(query); err != nil {
		return err
	}

	// Add new columns if they don't exist (for migration)
	migrations := []string{
		"ALTER TABLE streamer_status ADD COLUMN last_query_failed INTEGER DEFAULT 0",
		"ALTER TABLE streamer_status ADD COLUMN avatar_url TEXT",
		"ALTER TABLE streamer_status ADD COLUMN avatar_local TEXT",
		"ALTER TABLE streamer_status ADD COLUMN avatar_updated DATETIME",
	}

	for _, m := range migrations {
		// Ignore errors (column may already exist)
		db.conn.Exec(m)
	}

	return nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// StartSession creates a new live session record
func (db *DB) StartSession(streamerID string, platform model.Platform, roomID, title string) (int64, error) {
	result, err := db.conn.Exec(
		"INSERT INTO live_sessions (streamer_id, platform, room_id, title, start_time) VALUES (?, ?, ?, ?, ?)",
		streamerID, platform, roomID, title, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// EndSession marks a session as ended
func (db *DB) EndSession(sessionID int64) error {
	_, err := db.conn.Exec(
		"UPDATE live_sessions SET end_time = ? WHERE id = ?",
		time.Now(), sessionID,
	)
	return err
}

// GetActiveSession returns the current active session for a streamer (if any)
func (db *DB) GetActiveSession(streamerID string) (*model.LiveSession, error) {
	row := db.conn.QueryRow(
		"SELECT id, streamer_id, platform, room_id, title, start_time FROM live_sessions WHERE streamer_id = ? AND end_time IS NULL ORDER BY start_time DESC LIMIT 1",
		streamerID,
	)

	var session model.LiveSession
	var platform string
	err := row.Scan(&session.ID, &session.StreamerID, &platform, &session.RoomID, &session.Title, &session.StartTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	session.Platform = model.Platform(platform)
	return &session, nil
}

// GetHistory returns live session history for a streamer
func (db *DB) GetHistory(streamerID string, limit int) ([]model.LiveSession, error) {
	rows, err := db.conn.Query(
		"SELECT id, streamer_id, platform, room_id, title, start_time, end_time FROM live_sessions WHERE streamer_id = ? ORDER BY start_time DESC LIMIT ?",
		streamerID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []model.LiveSession
	for rows.Next() {
		var s model.LiveSession
		var platform string
		var endTime sql.NullTime
		if err := rows.Scan(&s.ID, &s.StreamerID, &platform, &s.RoomID, &s.Title, &s.StartTime, &endTime); err != nil {
			return nil, err
		}
		s.Platform = model.Platform(platform)
		if endTime.Valid {
			s.EndTime = &endTime.Time
			s.Duration = int64(endTime.Time.Sub(s.StartTime).Seconds())
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// GetStats returns statistics for a streamer
func (db *DB) GetStats(streamerID string) (*model.StreamerStats, error) {
	stats := &model.StreamerStats{StreamerID: streamerID}

	// Total sessions and duration
	row := db.conn.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(CASE WHEN end_time IS NOT NULL THEN strftime('%s', end_time) - strftime('%s', start_time) ELSE 0 END), 0)
		FROM live_sessions WHERE streamer_id = ?
	`, streamerID)
	if err := row.Scan(&stats.TotalSessions, &stats.TotalDuration); err != nil {
		return nil, err
	}

	if stats.TotalSessions > 0 {
		stats.AvgDuration = stats.TotalDuration / int64(stats.TotalSessions)
	}

	// Last live time
	row = db.conn.QueryRow(
		"SELECT start_time FROM live_sessions WHERE streamer_id = ? ORDER BY start_time DESC LIMIT 1",
		streamerID,
	)
	var lastTime time.Time
	if err := row.Scan(&lastTime); err == nil {
		stats.LastLiveTime = lastTime.Format("2006-01-02 15:04:05")
	}

	// Week sessions
	weekAgo := time.Now().AddDate(0, 0, -7)
	row = db.conn.QueryRow(
		"SELECT COUNT(*) FROM live_sessions WHERE streamer_id = ? AND start_time >= ?",
		streamerID, weekAgo,
	)
	row.Scan(&stats.WeekSessions)

	// Month sessions
	monthAgo := time.Now().AddDate(0, -1, 0)
	row = db.conn.QueryRow(
		"SELECT COUNT(*) FROM live_sessions WHERE streamer_id = ? AND start_time >= ?",
		streamerID, monthAgo,
	)
	row.Scan(&stats.MonthSessions)

	return stats, nil
}

// GetLastQueryTime returns the last query time for a streamer
func (db *DB) GetLastQueryTime(streamerID string) (*time.Time, error) {
	row := db.conn.QueryRow(
		"SELECT last_query_time FROM streamer_status WHERE streamer_id = ?",
		streamerID,
	)
	var lastTime sql.NullTime
	err := row.Scan(&lastTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !lastTime.Valid {
		return nil, nil
	}
	return &lastTime.Time, nil
}

// UpdateStreamerStatus updates the streamer's last query time and status
func (db *DB) UpdateStreamerStatus(streamerID string, isLive bool, title string, viewerCount int64, failed bool) error {
	failedInt := 0
	if failed {
		failedInt = 1
	}
	_, err := db.conn.Exec(`
		INSERT INTO streamer_status (streamer_id, last_query_time, last_query_failed, is_live, title, viewer_count)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(streamer_id) DO UPDATE SET
			last_query_time = excluded.last_query_time,
			last_query_failed = excluded.last_query_failed,
			is_live = excluded.is_live,
			title = excluded.title,
			viewer_count = excluded.viewer_count
	`, streamerID, time.Now(), failedInt, isLive, title, viewerCount)
	return err
}

// GetStreamerStatus returns the cached status for a streamer
func (db *DB) GetStreamerStatus(streamerID string) (lastQueryTime *time.Time, lastQueryFailed bool, isLive bool, title string, viewerCount int64, avatarLocal string, err error) {
	row := db.conn.QueryRow(
		"SELECT last_query_time, COALESCE(last_query_failed, 0), is_live, title, viewer_count, COALESCE(avatar_local, '') FROM streamer_status WHERE streamer_id = ?",
		streamerID,
	)
	var lastTime sql.NullTime
	var failedInt, liveInt int
	var titleNull sql.NullString
	err = row.Scan(&lastTime, &failedInt, &liveInt, &titleNull, &viewerCount, &avatarLocal)
	if err == sql.ErrNoRows {
		return nil, false, false, "", 0, "", nil
	}
	if err != nil {
		return nil, false, false, "", 0, "", err
	}
	if lastTime.Valid {
		lastQueryTime = &lastTime.Time
	}
	lastQueryFailed = failedInt == 1
	isLive = liveInt == 1
	if titleNull.Valid {
		title = titleNull.String
	}
	return
}

// GetAvatarInfo returns avatar info for a streamer
func (db *DB) GetAvatarInfo(streamerID string) (avatarURL, avatarLocal string, avatarUpdated *time.Time, err error) {
	row := db.conn.QueryRow(
		"SELECT COALESCE(avatar_url, ''), COALESCE(avatar_local, ''), avatar_updated FROM streamer_status WHERE streamer_id = ?",
		streamerID,
	)
	var updated sql.NullTime
	err = row.Scan(&avatarURL, &avatarLocal, &updated)
	if err == sql.ErrNoRows {
		return "", "", nil, nil
	}
	if err != nil {
		return "", "", nil, err
	}
	if updated.Valid {
		avatarUpdated = &updated.Time
	}
	return
}

// UpdateAvatar updates the avatar info for a streamer
func (db *DB) UpdateAvatar(streamerID, avatarURL, avatarLocal string) error {
	_, err := db.conn.Exec(`
		INSERT INTO streamer_status (streamer_id, avatar_url, avatar_local, avatar_updated)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(streamer_id) DO UPDATE SET
			avatar_url = excluded.avatar_url,
			avatar_local = excluded.avatar_local,
			avatar_updated = excluded.avatar_updated
	`, streamerID, avatarURL, avatarLocal, time.Now())
	return err
}
