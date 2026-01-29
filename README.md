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

- [ ] Docker deployment
- [ ] Notification integration (Telegram, Discord, etc.)
- [ ] Distributed crawling triggered by visitors

## Credits

Streamer list data from: [抽象赛道网](http://175.178.29.106/)
