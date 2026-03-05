// main.js — application entry point.
// Owns app state, wires modules together, and sets up all event listeners.

import * as api    from './api.js';
import * as ws     from './ws.js';
import * as pomo   from './pomodoro.js';
import * as render from './render.js';

// ---- App state ----
let briefingData = null;
let todoFilter   = 'all';

// ---- Boot ----
document.addEventListener('DOMContentLoaded', () => {
    setHeaderDate();
    pomo.init();
    pomo.bindControls();
    setupEventListeners();
    ws.connect(handleWsUpdate, updateConnectionStatus);
    loadBriefing();
    setInterval(renderTimeSensitive, 60_000);
});

// ---- Data loading ----
async function loadBriefing() {
    try {
        briefingData = await api.fetchBriefing();
        renderAll();
    } catch (e) {
        console.error('Briefing fetch error:', e);
    }
}

// ---- WebSocket updates ----
function handleWsUpdate(update) {
    if (!briefingData) return;
    switch (update.type) {
        case 'full_refresh':
            briefingData = update.payload;
            renderAll();
            break;
        case 'todos_updated':
            briefingData.todos = update.payload;
            render.renderTodos(briefingData.todos, todoFilter);
            render.renderScores(briefingData);
            break;
        case 'weather_updated':
            briefingData.weather = update.payload;
            render.renderWeather(briefingData.weather);
            break;
        case 'emails_updated':
            briefingData.unread_emails = update.payload;
            render.renderEmails(briefingData.unread_emails);
            render.renderScores(briefingData);
            break;
        case 'slack_updated':
            briefingData.slack_messages = update.payload;
            render.renderSlack(briefingData.slack_messages);
            render.renderScores(briefingData);
            break;
    }
}

function updateConnectionStatus(connected) {
    document.getElementById('ws-status').classList.toggle('connected', connected);
    document.getElementById('ws-label').textContent = connected ? 'Live' : 'Reconnecting...';
}

// ---- Render ----
function renderAll() {
    if (!briefingData) return;
    render.renderWeather(briefingData.weather);
    render.renderCalendar(briefingData.events);
    render.renderNews(briefingData.news);
    render.renderEmails(briefingData.unread_emails);
    render.renderSlack(briefingData.slack_messages);
    render.renderGitHub(briefingData.github_notifications);
    render.renderTodos(briefingData.todos, todoFilter);
    render.renderScores(briefingData);
}

function renderTimeSensitive() {
    if (!briefingData) return;
    render.renderCalendar(briefingData.events);
    render.renderSlack(briefingData.slack_messages);
    render.renderEmails(briefingData.unread_emails);
}

// ---- Event listeners ----
function setupEventListeners() {
    // Todo: add
    document.getElementById('todo-input').addEventListener('keypress', e => {
        if (e.key === 'Enter') handleAddTodo();
    });
    document.querySelector('.todo-add-btn').addEventListener('click', handleAddTodo);

    // Todo: toggle / delete (event delegation on the list container)
    document.getElementById('todo-list').addEventListener('click', async (e) => {
        const actionEl = e.target.closest('[data-action]');
        const itemEl   = e.target.closest('[data-todo-id]');
        if (!actionEl || !itemEl) return;

        const id     = itemEl.dataset.todoId;
        const status = Number(itemEl.dataset.todoStatus);

        try {
            if (actionEl.dataset.action === 'toggle') {
                briefingData.todos = await api.toggleTodo(id, status);
            } else if (actionEl.dataset.action === 'delete') {
                briefingData.todos = await api.deleteTodo(id);
            }
            render.renderTodos(briefingData.todos, todoFilter);
            render.renderScores(briefingData);
        } catch (err) {
            console.error('Todo action error:', err);
        }
    });

    // Todo: filter tabs
    document.querySelector('.todo-filters').addEventListener('click', e => {
        const btn = e.target.closest('[data-filter]');
        if (!btn) return;
        todoFilter = btn.dataset.filter;
        document.querySelectorAll('.todo-filter').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        render.renderTodos(briefingData?.todos || [], todoFilter);
    });

    // News / GitHub: open URLs (event delegation)
    document.getElementById('news-list').addEventListener('click', openUrlOnClick);
    document.getElementById('github-list').addEventListener('click', openUrlOnClick);
}

async function handleAddTodo() {
    const input = document.getElementById('todo-input');
    const title = input.value.trim();
    if (!title) return;
    input.value = '';
    try {
        briefingData.todos = await api.createTodo(title);
        render.renderTodos(briefingData.todos, todoFilter);
        render.renderScores(briefingData);
    } catch (e) {
        console.error('Add todo error:', e);
    }
}

function openUrlOnClick(e) {
    const item = e.target.closest('[data-url]');
    if (item?.dataset.url) window.open(item.dataset.url, '_blank');
}

// ---- Header date ----
function setHeaderDate() {
    const opts = { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' };
    document.getElementById('header-date').textContent =
        new Date().toLocaleDateString('en-US', opts);
}
