# 抽象赛道⏰

抽象TV开播提醒⏰ 赛道实在太多🌶️ 

## Features

- Supports 6 platforms: Bilibili, Douyin, Kuaishou, Douyu, NetEase CC, Weibo
- Auto-scans live status and records streaming history
- Statistics: total sessions, duration, weekly/monthly data
- Local avatar caching (no third-party requests from frontend)
- Modern dark theme UI

## Quick Start

```bash
# Install dependencies
go mod tidy

# Build
go build -o cxtv-alerts .

# Run
./cxtv-alerts
```

Visit http://localhost:8080

## Docker Deployment

```bash
# Create directories for persistent data
mkdir -p config data avatars

# Copy config files (edit as needed)
curl -o config/settings.json https://raw.githubusercontent.com/posoo/cxtv-alerts/main/config/settings.json
curl -o config/streamers.json https://raw.githubusercontent.com/posoo/cxtv-alerts/main/config/streamers.json

# Run with docker-compose
docker-compose up -d
```

Or run directly with Docker:

```bash
docker run -d \
  --name cxtv-alerts \
  -p 8080:8080 \
  -v ./config:/app/config \
  -v ./data:/app/data \
  -v ./avatars:/app/web/avatars \
  ghcr.io/posoo/cxtv-alerts:latest
```

## Configuration

### `config/settings.json`

```json
{
  "scan_interval_minutes": 5,
  "platform_delay_min_seconds": 5,
  "platform_delay_max_seconds": 20
}
```

### `config/streamers.json`

Streamer list configuration. See existing file for format.

## TODO

- [x] Docker deployment
- [ ] Notification integration (Telegram, Discord, etc.)
- [ ] Distributed crawling triggered by visitors

## Credits

Streamer list data from: [抽象赛道网](http://175.178.29.106/)

## License

[WTFPL](LICENSE)
