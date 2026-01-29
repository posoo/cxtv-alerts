# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/claude-code) when working with this repository.

## Project Overview

Live Streaming Alerts - A live streaming alert website that monitors streamers across multiple Chinese platforms.

## Tech Stack

- **Backend**: Go + Gin framework
- **Database**: SQLite
- **Frontend**: Vanilla HTML/CSS/JS (dark theme)

## Project Structure

```
cxtv-alerts/
├── main.go                    # Entry point
├── config/
│   ├── settings.json          # Scan intervals and delays
│   └── streamers.json         # Streamer list
├── internal/
│   ├── crawler/               # Platform-specific crawlers
│   │   ├── bilibili.go        # Bilibili (API)
│   │   ├── douyin.go          # Douyin (HTML scraping)
│   │   ├── kuaishou.go        # Kuaishou (HTML scraping)
│   │   ├── douyu.go           # Douyu (API)
│   │   ├── cc163.go           # NetEase CC (HTML scraping)
│   │   └── weibo.go           # Weibo (HTML scraping)
│   ├── database/              # SQLite operations
│   ├── handler/               # HTTP handlers
│   ├── model/                 # Data models
│   └── service/               # Business logic & avatar downloader
└── web/
    ├── index.html
    ├── style.css
    ├── app.js
    └── avatars/               # Cached avatar images
```

## Key Design Decisions

1. **Anti-crawling**: Configurable delays between requests (5-20s random), platform-specific rate limiting
2. **Local avatars**: Downloaded daily, no third-party requests from frontend
3. **Query status tracking**: Shows last query time and failure status per streamer
4. **Database migration**: Auto-adds new columns to existing databases

## Common Commands

```bash
# Build
go build -o cxtv-alerts .

# Run
./cxtv-alerts

# Clean database (if schema changes)
rm data.db
```

## API Endpoints

- `GET /api/streamers` - Get all streamers with live status
- `GET /api/history/:id` - Get streaming history for a streamer
- `GET /api/stats/:id` - Get statistics for a streamer
