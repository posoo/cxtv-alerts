const platformNames = {
    bilibili: 'B站',
    douyu: '斗鱼',
    douyin: '抖音',
    kuaishou: '快手',
    cc163: '网易CC',
    weibo: '微博'
};

let streamers = [];

async function fetchStreamers() {
    try {
        const response = await fetch('/api/streamers');
        const result = await response.json();
        if (result.code === 0) {
            streamers = result.data;
            renderStreamers();
            updateStats();
        }
    } catch (error) {
        console.error('Error fetching streamers:', error);
    }
}

function renderStreamers() {
    const grid = document.getElementById('streamersGrid');

    if (!streamers || streamers.length === 0) {
        grid.innerHTML = '<div class="loading">暂无主播数据</div>';
        return;
    }

    grid.innerHTML = streamers.map(s => {
        const avatarSrc = s.avatar_local || s.avatar;
        return `
        <div class="streamer-card ${s.is_live ? 'live' : ''}" data-id="${s.id}">
            <div class="card-header">
                ${avatarSrc
                    ? `<img class="avatar" src="${avatarSrc}" alt="${escapeHtml(s.name)}" onerror="this.outerHTML='<div class=\\'avatar-placeholder\\'>${escapeHtml(s.name.charAt(0))}</div>'">`
                    : `<div class="avatar-placeholder">${escapeHtml(s.name.charAt(0))}</div>`
                }
                <div class="streamer-info">
                    <div class="streamer-name">${escapeHtml(s.name)}</div>
                    <span class="platform-badge platform-${s.platform}">${platformNames[s.platform] || s.platform}</span>
                </div>
                ${s.is_live
                    ? `<div class="live-indicator">
                         <span class="live-dot"></span>
                         <span class="live-text">直播中</span>
                       </div>`
                    : `<span class="offline-text">未开播</span>`
                }
            </div>
            <div class="card-body">
                ${s.is_live ? `
                    <div class="stream-title" title="${escapeHtml(s.title || '')}">${escapeHtml(s.title || '无标题')}</div>
                    <div class="stream-meta">
                        <span class="viewer-count">👁 ${formatNumber(s.viewer_count || 0)}</span>
                        <span>${s.start_time ? '开播: ' + parseUTCTimestamp(s.start_time).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) : ''}</span>
                    </div>
                ` : ''}
                <div class="card-footer">
                    <span class="last-query ${s.last_query_failed ? 'query-failed' : ''}" title="最后查询时间${s.last_query_failed ? ' (查询失败)' : ''}">
                        ${s.last_query_failed ? '⚠️' : '🕐'} ${s.last_query_time ? formatQueryTime(s.last_query_time) : '未查询'}${s.last_query_failed ? ' 失败' : ''}
                    </span>
                    <div class="card-actions">
                        <button class="btn-stats" onclick="event.stopPropagation(); showStats('${s.id}', '${escapeHtml(s.name)}')">统计</button>
                        ${s.room_url ? `<a class="btn-open" href="${s.room_url}" target="_blank" onclick="event.stopPropagation()">打开直播间</a>` : ''}
                    </div>
                </div>
            </div>
        </div>
    `}).join('');
}

// Parse UTC timestamp string and convert to local Date object
function parseUTCTimestamp(timeStr) {
    if (!timeStr) return null;
    // Append 'Z' to indicate UTC timezone
    return new Date(timeStr.replace(' ', 'T') + 'Z');
}

function formatQueryTime(timeStr) {
    if (!timeStr) return '未查询';
    const date = parseUTCTimestamp(timeStr);
    const now = new Date();
    const diff = Math.floor((now - date) / 1000);

    if (diff < 60) return `${diff}秒前`;
    if (diff < 3600) return `${Math.floor(diff / 60)}分钟前`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}小时前`;
    // Convert to local time for display
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
}

function updateStats() {
    const total = streamers.length;
    const live = streamers.filter(s => s.is_live).length;

    document.getElementById('totalCount').textContent = total;
    document.getElementById('liveCount').textContent = live;
    document.getElementById('lastUpdate').textContent = new Date().toLocaleTimeString('zh-CN');
}

async function showStats(id, name) {
    const modal = document.getElementById('statsModal');
    const modalTitle = document.getElementById('modalTitle');
    const modalBody = document.getElementById('modalBody');

    modalTitle.textContent = `${name} - 统计数据`;
    modalBody.innerHTML = '<div class="loading">加载中...</div>';
    modal.classList.add('show');

    try {
        const [statsRes, historyRes] = await Promise.all([
            fetch(`/api/stats/${id}`),
            fetch(`/api/history/${id}?limit=10`)
        ]);

        const stats = await statsRes.json();
        const history = await historyRes.json();

        if (stats.code !== 0 || history.code !== 0) {
            throw new Error('Failed to fetch data');
        }

        const s = stats.data;
        const h = history.data || [];

        modalBody.innerHTML = `
            <div class="stats-grid">
                <div class="stats-item">
                    <div class="value">${s.total_sessions}</div>
                    <div class="label">总开播次数</div>
                </div>
                <div class="stats-item">
                    <div class="value">${formatDuration(s.total_duration)}</div>
                    <div class="label">总直播时长</div>
                </div>
                <div class="stats-item">
                    <div class="value">${s.week_sessions}</div>
                    <div class="label">本周开播</div>
                </div>
                <div class="stats-item">
                    <div class="value">${s.month_sessions}</div>
                    <div class="label">本月开播</div>
                </div>
            </div>
            ${s.last_live_time ? `<p style="color: var(--text-secondary); margin-bottom: 1rem;">上次开播时间: ${parseUTCTimestamp(s.last_live_time).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })}</p>` : ''}
            <div class="history-section">
                <h3>近期开播记录</h3>
                ${h.length > 0 ? `
                    <div class="history-list">
                        ${h.map(item => `
                            <div class="history-item">
                                <div class="title">${escapeHtml(item.title || '无标题')}</div>
                                <div class="meta">
                                    <span>${formatDateTime(item.start_time)}</span>
                                    <span>${item.duration ? formatDuration(item.duration) : '进行中'}</span>
                                </div>
                            </div>
                        `).join('')}
                    </div>
                ` : '<p style="color: var(--text-secondary);">暂无开播记录</p>'}
            </div>
        `;
    } catch (error) {
        console.error('Error fetching stats:', error);
        modalBody.innerHTML = '<div class="loading">加载失败</div>';
    }
}

function closeModal() {
    document.getElementById('statsModal').classList.remove('show');
}

function escapeHtml(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

function formatNumber(num) {
    if (num >= 10000) {
        return (num / 10000).toFixed(1) + '万';
    }
    return num.toString();
}

function formatDuration(seconds) {
    if (!seconds) return '0分钟';
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (hours > 0) {
        return `${hours}小时${minutes}分钟`;
    }
    return `${minutes}分钟`;
}

function formatDateTime(timeStr) {
    if (!timeStr) return '';
    const date = parseUTCTimestamp(timeStr);
    return date.toLocaleString('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}

// Close modal on outside click
document.getElementById('statsModal').addEventListener('click', (e) => {
    if (e.target.id === 'statsModal') {
        closeModal();
    }
});

// Close modal on Escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        closeModal();
    }
});

// Initial fetch
fetchStreamers();

// Auto refresh every 30 seconds
setInterval(fetchStreamers, 30000);
