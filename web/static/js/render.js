// render.js — pure DOM renderers. Each function receives data and writes to the DOM.
// No fetch calls, no app state, no event listeners.

import { escapeHtml, timeAgo, formatTime, formatDate, capitalize } from './utils.js';
import { TODO_STATUS, PRIORITY } from './api.js';

const PRIORITY_CLASS = ['priority-low', 'priority-medium', 'priority-high', 'priority-urgent'];
const PRIORITY_LABEL = ['low', 'med', 'high', 'urgent'];

const JIRA_PRIORITY_COLOR = {
    Critical: 'var(--accent-red)',
    High:     'var(--accent-orange)',
    Medium:   'var(--accent-yellow)',
    Low:      'var(--text-dim)',
};

const NOTION_STATUS_COLOR = {
    'In Progress': 'var(--accent-blue)',
    'Not Started': 'var(--text-dim)',
    Done:          'var(--accent-green)',
};

// ---- Scores ----

export function renderScores(briefing) {
    const emails    = briefing.unread_emails || [];
    const unreadE   = emails.filter(e => e.is_unread).length;
    const slackMsgs = (briefing.slack_messages || []).length;
    const ghNotifs  = (briefing.github_notifications || []).filter(n => n.unread).length;
    const events    = (briefing.events || []).length;
    const todos     = briefing.todos || [];
    const pending   = todos.filter(t => t.status === TODO_STATUS.PENDING || t.status === TODO_STATUS.IN_PROGRESS).length;
    const done      = todos.filter(t => t.status === TODO_STATUS.DONE).length;

    setText('score-emails', unreadE);
    setText('score-slack', slackMsgs);
    setText('score-github', ghNotifs);
    setText('score-events', events);
    setText('score-todos', pending);

    const total    = emails.length + slackMsgs + (briefing.github_notifications || []).length + pending + done;
    const handled  = total - (unreadE + slackMsgs + ghNotifs + pending);
    const score    = total > 0 ? Math.round((handled / total) * 100) : 100;
    setText('score-inbox-zero', score + '%');

    // Progress bars
    setProgress('prog-email',  pct(emails.filter(e => !e.is_unread).length, emails.length));
    setProgress('prog-slack',  0);
    setProgress('prog-github', pct((briefing.github_notifications || []).filter(n => !n.unread).length, (briefing.github_notifications || []).length));
    setProgress('prog-todos',  pct(done, pending + done));
}

// ---- Weather ----

export function renderWeather(weather) {
    if (!weather) return;
    setText('weather-temp', `${Math.round(weather.temperature)}°`);
    setText('weather-desc', capitalize(weather.description));
    setHTML('weather-details',
        `<span>Feels like ${Math.round(weather.feels_like)}°</span>` +
        `<span>Humidity ${weather.humidity}%</span>` +
        `<span>Wind ${Math.round(weather.wind_speed)} mph</span>`);
}

// ---- Calendar ----

export function renderCalendar(events) {
    setText('event-count', (events || []).length);
    if (!events || events.length === 0) {
        setHTML('events-list', empty('No events today'));
        return;
    }
    const now = new Date();
    setHTML('events-list', events.map(e => {
        const start   = new Date(e.start_time);
        const end     = new Date(e.end_time);
        const isNow   = now >= start && now <= end;
        const timeStr = e.all_day ? 'All day' : `${formatTime(start)} – ${formatTime(end)}`;
        const loc     = e.location ? ` &middot; ${escapeHtml(e.location)}` : '';
        const join    = e.meeting_url
            ? ` <a href="${escapeHtml(e.meeting_url)}" target="_blank" style="color:var(--accent-blue);font-size:11px;">Join</a>`
            : '';
        return `<div class="list-item ${isNow ? 'event-now' : ''}">
            <span class="event-time">${timeStr}</span>
            <div class="list-item-content">
                <div class="list-item-title">${escapeHtml(e.title)}${join}</div>
                <div class="list-item-meta">${e.attendees ? e.attendees.length + ' attendees' : ''}${loc}</div>
            </div>
        </div>`;
    }).join(''));
}

// ---- News ----

export function renderNews(news) {
    if (!news || news.length === 0) {
        setHTML('news-list', empty('No news available'));
        return;
    }
    setHTML('news-list', news.map(n => {
        const desc = n.description && n.description !== '#'
            ? `<div class="news-desc">${escapeHtml(n.description)}</div>` : '';
        const url  = n.url && n.url !== '#' ? escapeHtml(n.url) : '';
        return `<div class="news-item${url ? ' clickable' : ''}" ${url ? `data-url="${url}"` : ''}>
            <div class="list-item-title">${escapeHtml(n.title)}</div>
            ${desc}
            <div class="list-item-meta">${escapeHtml(n.source)} &middot; ${timeAgo(n.published_at)}</div>
        </div>`;
    }).join(''));
}

// ---- Email ----

export function renderEmails(emails) {
    const unread = (emails || []).filter(e => e.is_unread).length;
    setText('email-count', unread);
    if (!emails || emails.length === 0) {
        setHTML('email-list', empty('Inbox zero! Nice work.'));
        return;
    }
    setHTML('email-list', emails.map(e => {
        const star = e.is_starred ? '<span style="color:var(--accent-yellow)">&#9733; </span>' : '';
        const badge = e.is_unread ? badge_html('new', 'var(--accent-blue)') : '';
        return `<div class="list-item">
            <div class="list-item-content">
                <div class="list-item-title">${star}${escapeHtml(e.subject)}</div>
                <div class="list-item-meta">${escapeHtml(e.from)} &middot; ${timeAgo(e.date)}</div>
            </div>
            ${badge}
        </div>`;
    }).join(''));
}

// ---- Slack ----

export function renderSlack(messages) {
    setText('slack-count', (messages || []).length);
    if (!messages || messages.length === 0) {
        setHTML('slack-list', empty('No new messages'));
        return;
    }
    setHTML('slack-list', messages.map(m => {
        const urgentBadge = m.is_urgent ? badge_html('urgent', 'var(--accent-red)') : '';
        const dmBadge     = m.is_dm    ? badge_html('DM', 'var(--accent-purple)') : '';
        return `<div class="list-item">
            <div class="list-item-content">
                <div class="list-item-title">${escapeHtml(m.user)} <span style="color:var(--text-dim)">${escapeHtml(m.channel)}</span></div>
                <div class="list-item-meta">${escapeHtml(m.text)}</div>
            </div>
            <div style="display:flex;gap:4px;flex-shrink:0;">${urgentBadge}${dmBadge}</div>
            <span class="list-item-time">${timeAgo(m.timestamp)}</span>
        </div>`;
    }).join(''));
}

// ---- GitHub ----

export function renderGitHub(notifications) {
    const unread = (notifications || []).filter(n => n.unread).length;
    setText('github-count', unread);
    if (!notifications || notifications.length === 0) {
        setHTML('github-list', empty('No notifications'));
        return;
    }
    setHTML('github-list', notifications.map(n => {
        const isPR    = n.type === 'PullRequest';
        const label   = isPR ? 'PR' : 'IS';
        const color   = isPR ? 'var(--accent-green)' : 'var(--accent-purple)';
        const newBadge = n.unread ? badge_html('new', 'var(--accent-blue)') : '';
        return `<div class="list-item">
            <div class="list-item-icon" style="background:${color}20;color:${color};font-size:11px;font-weight:700;">${label}</div>
            <div class="list-item-content">
                <div class="list-item-title">${escapeHtml(n.title)}</div>
                <div class="list-item-meta">${escapeHtml(n.repo)} &middot; ${escapeHtml(n.reason)} &middot; ${timeAgo(n.updated_at)}</div>
            </div>
            ${newBadge}
        </div>`;
    }).join(''));
}

// ---- Todos ----

export function renderTodos(todos, activeFilter) {
    todos = todos || [];
    const pending = todos.filter(t => t.status === TODO_STATUS.PENDING || t.status === TODO_STATUS.IN_PROGRESS).length;
    setText('todo-count', pending);

    let filtered = applyFilter(todos, activeFilter);
    filtered = sortTodos(filtered);

    if (filtered.length === 0) {
        const msg = activeFilter === 'done' ? 'No completed tasks yet' : 'All clear! Add a task above.';
        setHTML('todo-list', empty(msg));
        return;
    }

    setHTML('todo-list', filtered.map(t => {
        const isDone    = t.status === TODO_STATUS.DONE;
        const priClass  = PRIORITY_CLASS[t.priority] || 'priority-medium';
        const priLabel  = PRIORITY_LABEL[t.priority] || 'med';
        const source    = t.source || 'manual';
        const dueStr    = t.due_date
            ? `<span style="font-size:10px;color:var(--accent-orange);">due ${formatDate(t.due_date)}</span>` : '';
        return `<div class="todo-item ${priClass}" data-todo-id="${t.id}" data-todo-status="${t.status}">
            <div class="todo-checkbox ${isDone ? 'done' : ''}" data-action="toggle"></div>
            <div class="todo-content">
                <div class="todo-title ${isDone ? 'done' : ''}">${escapeHtml(t.title)}</div>
                <div class="todo-meta">
                    <span class="source-badge ${source}">${source}</span>
                    <span style="font-size:10px;color:var(--text-dim);">${priLabel}</span>
                    ${dueStr}
                </div>
            </div>
            <button class="todo-delete" data-action="delete" title="Delete">&times;</button>
        </div>`;
    }).join(''));
}

function applyFilter(todos, filter) {
    switch (filter) {
        case 'pending': return todos.filter(t => t.status === TODO_STATUS.PENDING || t.status === TODO_STATUS.IN_PROGRESS);
        case 'done':    return todos.filter(t => t.status === TODO_STATUS.DONE);
        case 'urgent':  return todos.filter(t => t.priority >= PRIORITY.HIGH && t.status !== TODO_STATUS.DONE);
        default:        return todos;
    }
}

function sortTodos(todos) {
    return [...todos].sort((a, b) => {
        if (a.status === TODO_STATUS.DONE && b.status !== TODO_STATUS.DONE) return 1;
        if (a.status !== TODO_STATUS.DONE && b.status === TODO_STATUS.DONE) return -1;
        if (b.priority !== a.priority) return b.priority - a.priority;
        return new Date(b.created_at) - new Date(a.created_at);
    });
}

// ---- Helpers ----

function setText(id, value) {
    const el = document.getElementById(id);
    if (el) el.textContent = value;
}

function setHTML(id, html) {
    const el = document.getElementById(id);
    if (el) el.innerHTML = html;
}

function setProgress(id, percent) {
    const bar = document.getElementById(id);
    const val = document.getElementById(id + '-val');
    if (bar) bar.style.width = percent + '%';
    if (val) val.textContent = percent + '%';
}

function pct(part, total) {
    return total > 0 ? Math.round((part / total) * 100) : 100;
}

function empty(msg) {
    return `<div class="empty-state">${msg}</div>`;
}

function badge_html(label, color) {
    return `<span class="card-badge" style="background:${color};font-size:10px;">${label}</span>`;
}
