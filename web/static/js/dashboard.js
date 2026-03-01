// ============================================================
// Daily Briefing Dashboard — Real-time Interactive Frontend
// ============================================================

let ws = null;
let briefingData = null;
let todoFilter = 'all';
let pomodoroInterval = null;
let pomodoroSeconds = 25 * 60;
let pomodoroRunning = false;
let pomodoroIsBreak = false;

// ---- Init ----
document.addEventListener('DOMContentLoaded', () => {
    setHeaderDate();
    connectWebSocket();
    fetchBriefing();
    setInterval(updateRelativeTimes, 60000);
});

function setHeaderDate() {
    const now = new Date();
    const opts = { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' };
    document.getElementById('header-date').textContent = now.toLocaleDateString('en-US', opts);
}

// ---- WebSocket ----
function connectWebSocket() {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${proto}//${location.host}/ws`);

    ws.onopen = () => {
        document.getElementById('ws-status').classList.add('connected');
        document.getElementById('ws-label').textContent = 'Live';
    };

    ws.onclose = () => {
        document.getElementById('ws-status').classList.remove('connected');
        document.getElementById('ws-label').textContent = 'Reconnecting...';
        setTimeout(connectWebSocket, 3000);
    };

    ws.onmessage = (event) => {
        try {
            const update = JSON.parse(event.data);
            handleUpdate(update);
        } catch (e) {
            console.error('WS parse error:', e);
        }
    };
}

function handleUpdate(update) {
    switch (update.type) {
        case 'full_refresh':
            briefingData = update.payload;
            renderAll();
            break;
        case 'todos_updated':
            if (briefingData) {
                briefingData.todos = update.payload;
                renderTodos();
                updateScores();
            }
            break;
        case 'weather_updated':
            if (briefingData) {
                briefingData.weather = update.payload;
                renderWeather();
            }
            break;
        case 'emails_updated':
            if (briefingData) {
                briefingData.unread_emails = update.payload;
                renderEmails();
                updateScores();
            }
            break;
        case 'slack_updated':
            if (briefingData) {
                briefingData.slack_messages = update.payload;
                renderSlack();
                updateScores();
            }
            break;
    }
}

// ---- API ----
async function fetchBriefing() {
    try {
        const resp = await fetch('/api/briefing');
        briefingData = await resp.json();
        renderAll();
    } catch (e) {
        console.error('Fetch error:', e);
    }
}

async function addTodo() {
    const input = document.getElementById('todo-input');
    const title = input.value.trim();
    if (!title) return;

    try {
        await fetch('/api/todos', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ title, priority: 1, status: 0 })
        });
        input.value = '';
        // Update will come via WebSocket
        const resp = await fetch('/api/todos');
        briefingData.todos = await resp.json();
        renderTodos();
        updateScores();
    } catch (e) {
        console.error('Add todo error:', e);
    }
}

async function toggleTodo(id, currentStatus) {
    const newStatus = currentStatus === 2 ? 0 : 2; // toggle done/pending
    try {
        await fetch(`/api/todos/${id}`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ status: newStatus })
        });
        const resp = await fetch('/api/todos');
        briefingData.todos = await resp.json();
        renderTodos();
        updateScores();
    } catch (e) {
        console.error('Toggle error:', e);
    }
}

async function deleteTodo(id) {
    try {
        await fetch(`/api/todos/${id}`, { method: 'DELETE' });
        const resp = await fetch('/api/todos');
        briefingData.todos = await resp.json();
        renderTodos();
        updateScores();
    } catch (e) {
        console.error('Delete error:', e);
    }
}

// ---- Render All ----
function renderAll() {
    if (!briefingData) return;
    renderWeather();
    renderCalendar();
    renderNews();
    renderEmails();
    renderSlack();
    renderGitHub();
    renderTodos();
    updateScores();
    updateInboxZero();
}

// ---- Weather ----
function renderWeather() {
    const w = briefingData.weather;
    if (!w) return;
    document.getElementById('weather-temp').textContent = `${Math.round(w.temperature)}°`;
    document.getElementById('weather-desc').textContent = capitalize(w.description);
    document.getElementById('weather-details').innerHTML =
        `<span>Feels like ${Math.round(w.feels_like)}°</span>` +
        `<span>Humidity ${w.humidity}%</span>` +
        `<span>Wind ${Math.round(w.wind_speed)} mph</span>`;
}

// ---- Calendar ----
function renderCalendar() {
    const events = briefingData.events || [];
    document.getElementById('event-count').textContent = events.length;

    if (events.length === 0) {
        document.getElementById('events-list').innerHTML = '<div class="empty-state">No events today</div>';
        return;
    }

    const now = new Date();
    const html = events.map(e => {
        const start = new Date(e.start_time);
        const end = new Date(e.end_time);
        const isNow = now >= start && now <= end;
        const timeStr = e.all_day ? 'All day' : `${formatTime(start)} - ${formatTime(end)}`;
        const loc = e.location ? ` &middot; ${escapeHtml(e.location)}` : '';
        const meetLink = e.meeting_url ?
            ` <a href="${escapeHtml(e.meeting_url)}" target="_blank" style="color:var(--accent-blue);font-size:11px;">Join</a>` : '';

        return `<div class="list-item ${isNow ? 'event-now' : ''}">
            <span class="event-time">${timeStr}</span>
            <div class="list-item-content">
                <div class="list-item-title">${escapeHtml(e.title)}${meetLink}</div>
                <div class="list-item-meta">${e.attendees ? e.attendees.length + ' attendees' : ''}${loc}</div>
            </div>
        </div>`;
    }).join('');

    document.getElementById('events-list').innerHTML = html;
}

// ---- News ----
function renderNews() {
    const news = briefingData.news || [];
    if (news.length === 0) {
        document.getElementById('news-list').innerHTML = '<div class="empty-state">No news available</div>';
        return;
    }

    const html = news.map(n => {
        const desc = n.description && n.description !== '#' ?
            `<div class="news-desc">${escapeHtml(n.description)}</div>` : '';
        const urlAttr = n.url && n.url !== '#' ?
            `onclick="window.open('${escapeHtml(n.url)}','_blank')"` : '';
        return `<div class="news-item" ${urlAttr}>
            <div class="list-item-title">${escapeHtml(n.title)}</div>
            ${desc}
            <div class="list-item-meta">${escapeHtml(n.source)} &middot; ${timeAgo(n.published_at)}</div>
        </div>`;
    }).join('');

    document.getElementById('news-list').innerHTML = html;
}

// ---- Email ----
function renderEmails() {
    const emails = briefingData.unread_emails || [];
    const unread = emails.filter(e => e.is_unread).length;
    document.getElementById('email-count').textContent = unread;

    if (emails.length === 0) {
        document.getElementById('email-list').innerHTML = '<div class="empty-state">Inbox zero! Nice work.</div>';
        return;
    }

    const html = emails.map(e => {
        const star = e.is_starred ? '<span style="color:var(--accent-yellow)">&#9733; </span>' : '';
        return `<div class="list-item">
            <div class="list-item-content">
                <div class="list-item-title">${star}${escapeHtml(e.subject)}</div>
                <div class="list-item-meta">${escapeHtml(e.from)} &middot; ${timeAgo(e.date)}</div>
            </div>
            ${e.is_unread ? '<span class="card-badge" style="background:var(--accent-blue);font-size:10px;">new</span>' : ''}
        </div>`;
    }).join('');

    document.getElementById('email-list').innerHTML = html;
}

// ---- Slack ----
function renderSlack() {
    const msgs = briefingData.slack_messages || [];
    document.getElementById('slack-count').textContent = msgs.length;

    if (msgs.length === 0) {
        document.getElementById('slack-list').innerHTML = '<div class="empty-state">No new messages</div>';
        return;
    }

    const html = msgs.map(m => {
        const urgentBadge = m.is_urgent ?
            '<span class="card-badge" style="background:var(--accent-red);font-size:10px;">urgent</span>' : '';
        const dmBadge = m.is_dm ?
            '<span class="card-badge" style="background:var(--accent-purple);font-size:10px;">DM</span>' : '';
        return `<div class="list-item">
            <div class="list-item-content">
                <div class="list-item-title">${escapeHtml(m.user)} <span style="color:var(--text-dim)">${escapeHtml(m.channel)}</span></div>
                <div class="list-item-meta">${escapeHtml(m.text)}</div>
            </div>
            <div style="display:flex;gap:4px;flex-shrink:0;">
                ${urgentBadge}${dmBadge}
            </div>
            <span class="list-item-time">${timeAgo(m.timestamp)}</span>
        </div>`;
    }).join('');

    document.getElementById('slack-list').innerHTML = html;
}

// ---- GitHub ----
function renderGitHub() {
    const notifs = briefingData.github_notifications || [];
    const unread = notifs.filter(n => n.unread).length;
    document.getElementById('github-count').textContent = unread;

    if (notifs.length === 0) {
        document.getElementById('github-list').innerHTML = '<div class="empty-state">No notifications</div>';
        return;
    }

    const html = notifs.map(n => {
        const typeIcon = n.type === 'PullRequest' ? 'PR' : 'Issue';
        const typeColor = n.type === 'PullRequest' ? 'var(--accent-green)' : 'var(--accent-purple)';
        return `<div class="list-item">
            <div class="list-item-icon" style="background:${typeColor}20;color:${typeColor};font-size:11px;font-weight:700;">${typeIcon}</div>
            <div class="list-item-content">
                <div class="list-item-title">${escapeHtml(n.title)}</div>
                <div class="list-item-meta">${escapeHtml(n.repo)} &middot; ${escapeHtml(n.reason)} &middot; ${timeAgo(n.updated_at)}</div>
            </div>
            ${n.unread ? '<span class="card-badge" style="background:var(--accent-blue);font-size:10px;">new</span>' : ''}
        </div>`;
    }).join('');

    document.getElementById('github-list').innerHTML = html;
}

// ---- Todos ----
function renderTodos() {
    const todos = briefingData.todos || [];
    const pending = todos.filter(t => t.status !== 2 && t.status !== 3).length;
    document.getElementById('todo-count').textContent = pending;

    let filtered = todos;
    switch (todoFilter) {
        case 'pending':
            filtered = todos.filter(t => t.status === 0 || t.status === 1);
            break;
        case 'done':
            filtered = todos.filter(t => t.status === 2);
            break;
        case 'urgent':
            filtered = todos.filter(t => t.priority >= 2 && t.status !== 2);
            break;
    }

    // Sort: urgent first, then by priority desc, then by created
    filtered.sort((a, b) => {
        if (a.status === 2 && b.status !== 2) return 1;
        if (a.status !== 2 && b.status === 2) return -1;
        if (b.priority !== a.priority) return b.priority - a.priority;
        return new Date(b.created_at) - new Date(a.created_at);
    });

    if (filtered.length === 0) {
        document.getElementById('todo-list').innerHTML =
            `<div class="empty-state">${todoFilter === 'done' ? 'No completed tasks yet' : 'All clear! Add a task above.'}</div>`;
        return;
    }

    const priorityClasses = ['priority-low', 'priority-medium', 'priority-high', 'priority-urgent'];
    const priorityLabels = ['low', 'med', 'high', 'urgent'];

    const html = filtered.map(t => {
        const isDone = t.status === 2;
        const priClass = priorityClasses[t.priority] || 'priority-medium';
        const priLabel = priorityLabels[t.priority] || 'med';
        const sourceClass = t.source || 'manual';

        return `<div class="todo-item ${priClass}">
            <div class="todo-checkbox ${isDone ? 'done' : ''}" onclick="toggleTodo('${t.id}', ${t.status})"></div>
            <div class="todo-content">
                <div class="todo-title ${isDone ? 'done' : ''}">${escapeHtml(t.title)}</div>
                <div class="todo-meta">
                    <span class="source-badge ${sourceClass}">${sourceClass}</span>
                    <span style="font-size:10px;color:var(--text-dim);">${priLabel}</span>
                    ${t.due_date ? `<span style="font-size:10px;color:var(--accent-orange);">due ${formatDate(t.due_date)}</span>` : ''}
                </div>
            </div>
            <button class="todo-delete" onclick="deleteTodo('${t.id}')" title="Delete">&times;</button>
        </div>`;
    }).join('');

    document.getElementById('todo-list').innerHTML = html;
}

function filterTodos(filter, btn) {
    todoFilter = filter;
    document.querySelectorAll('.todo-filter').forEach(b => b.classList.remove('active'));
    if (btn) btn.classList.add('active');
    renderTodos();
}

// ---- Scores & Inbox Zero ----
function updateScores() {
    if (!briefingData) return;

    const emails = briefingData.unread_emails || [];
    const unreadEmails = emails.filter(e => e.is_unread).length;
    const slackMsgs = (briefingData.slack_messages || []).length;
    const ghNotifs = (briefingData.github_notifications || []).filter(n => n.unread).length;
    const events = (briefingData.events || []).length;
    const todos = briefingData.todos || [];
    const pendingTodos = todos.filter(t => t.status === 0 || t.status === 1).length;
    const doneTodos = todos.filter(t => t.status === 2).length;
    const totalTodos = pendingTodos + doneTodos;

    document.getElementById('score-emails').textContent = unreadEmails;
    document.getElementById('score-slack').textContent = slackMsgs;
    document.getElementById('score-github').textContent = ghNotifs;
    document.getElementById('score-events').textContent = events;
    document.getElementById('score-todos').textContent = pendingTodos;

    // Inbox zero score: higher is better (% of items handled)
    const totalSignals = unreadEmails + slackMsgs + ghNotifs + pendingTodos;
    const maxSignals = emails.length + slackMsgs + (briefingData.github_notifications || []).length + totalTodos;
    const handledSignals = maxSignals - totalSignals;
    const inboxScore = maxSignals > 0 ? Math.round((handledSignals / maxSignals) * 100) : 100;
    document.getElementById('score-inbox-zero').textContent = inboxScore + '%';

    updateInboxZero();
}

function updateInboxZero() {
    if (!briefingData) return;

    const emails = briefingData.unread_emails || [];
    const totalE = emails.length;
    const readE = emails.filter(e => !e.is_unread).length;
    const emailPct = totalE > 0 ? Math.round((readE / totalE) * 100) : 100;

    const slackMsgs = briefingData.slack_messages || [];
    const slackPct = 0; // Slack messages don't have read state in our model

    const ghNotifs = briefingData.github_notifications || [];
    const totalGH = ghNotifs.length;
    const readGH = ghNotifs.filter(n => !n.unread).length;
    const ghPct = totalGH > 0 ? Math.round((readGH / totalGH) * 100) : 100;

    const todos = briefingData.todos || [];
    const totalT = todos.length;
    const doneT = todos.filter(t => t.status === 2).length;
    const todoPct = totalT > 0 ? Math.round((doneT / totalT) * 100) : 100;

    setProgress('prog-email', emailPct);
    setProgress('prog-slack', slackPct);
    setProgress('prog-github', ghPct);
    setProgress('prog-todos', todoPct);
}

function setProgress(id, pct) {
    const el = document.getElementById(id);
    const val = document.getElementById(id + '-val');
    if (el) el.style.width = pct + '%';
    if (val) val.textContent = pct + '%';
}

// ---- Pomodoro ----
function togglePomodoro() {
    if (pomodoroRunning) {
        clearInterval(pomodoroInterval);
        pomodoroRunning = false;
        document.getElementById('pomo-start').textContent = 'Start';
        document.getElementById('pomo-start').classList.remove('active');
    } else {
        pomodoroRunning = true;
        document.getElementById('pomo-start').textContent = 'Pause';
        document.getElementById('pomo-start').classList.add('active');
        pomodoroInterval = setInterval(() => {
            pomodoroSeconds--;
            updatePomodoroDisplay();
            if (pomodoroSeconds <= 0) {
                clearInterval(pomodoroInterval);
                pomodoroRunning = false;
                if (pomodoroIsBreak) {
                    pomodoroSeconds = 25 * 60;
                    pomodoroIsBreak = false;
                    document.getElementById('pomo-start').textContent = 'Start Work';
                } else {
                    pomodoroSeconds = 5 * 60;
                    pomodoroIsBreak = true;
                    document.getElementById('pomo-start').textContent = 'Start Break';
                }
                document.getElementById('pomo-start').classList.remove('active');
                updatePomodoroDisplay();
                // Notification
                if (Notification.permission === 'granted') {
                    new Notification(pomodoroIsBreak ? 'Time for a break!' : 'Break over! Back to work.');
                }
            }
        }, 1000);
    }
}

function resetPomodoro() {
    clearInterval(pomodoroInterval);
    pomodoroRunning = false;
    pomodoroIsBreak = false;
    pomodoroSeconds = 25 * 60;
    document.getElementById('pomo-start').textContent = 'Start';
    document.getElementById('pomo-start').classList.remove('active');
    updatePomodoroDisplay();
}

function skipPomodoro() {
    clearInterval(pomodoroInterval);
    pomodoroRunning = false;
    if (pomodoroIsBreak) {
        pomodoroSeconds = 25 * 60;
        pomodoroIsBreak = false;
        document.getElementById('pomo-start').textContent = 'Start Work';
    } else {
        pomodoroSeconds = 5 * 60;
        pomodoroIsBreak = true;
        document.getElementById('pomo-start').textContent = 'Start Break';
    }
    document.getElementById('pomo-start').classList.remove('active');
    updatePomodoroDisplay();
}

function updatePomodoroDisplay() {
    const mins = Math.floor(pomodoroSeconds / 60);
    const secs = pomodoroSeconds % 60;
    document.getElementById('pomodoro-display').textContent =
        `${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
}

// ---- Utilities ----
function formatTime(dateStr) {
    const d = new Date(dateStr);
    return d.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit', hour12: true });
}

function formatDate(dateStr) {
    const d = new Date(dateStr);
    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

function timeAgo(dateStr) {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    const now = new Date();
    const diff = Math.floor((now - d) / 1000);
    if (diff < 60) return 'just now';
    if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
    if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
    return Math.floor(diff / 86400) + 'd ago';
}

function capitalize(s) {
    if (!s) return '';
    return s.charAt(0).toUpperCase() + s.slice(1);
}

function escapeHtml(s) {
    if (!s) return '';
    const div = document.createElement('div');
    div.appendChild(document.createTextNode(s));
    return div.innerHTML;
}

function updateRelativeTimes() {
    // Re-render time-sensitive sections
    if (briefingData) {
        renderCalendar();
        renderSlack();
        renderEmails();
    }
}

// Request notification permission for pomodoro
if ('Notification' in window && Notification.permission === 'default') {
    Notification.requestPermission();
}
