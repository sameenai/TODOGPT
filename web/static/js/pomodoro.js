// pomodoro.js — self-contained focus timer.
// Call init() once after DOMContentLoaded; bind buttons with bindControls().

const WORK_SECONDS  = 25 * 60;
const BREAK_SECONDS = 5 * 60;

let interval  = null;
let seconds   = WORK_SECONDS;
let running   = false;
let isBreak   = false;

export function init() {
    updateDisplay();
    if ('Notification' in window && Notification.permission === 'default') {
        Notification.requestPermission();
    }
}

export function bindControls() {
    document.getElementById('pomo-start').addEventListener('click', toggle);
    document.getElementById('pomo-reset').addEventListener('click', reset);
    document.getElementById('pomo-skip').addEventListener('click', skip);
}

export function toggle() {
    if (running) {
        pause();
    } else {
        start();
    }
}

function start() {
    running = true;
    setStartLabel('Pause');
    document.getElementById('pomo-start').classList.add('active');
    interval = setInterval(tick, 1000);
}

function pause() {
    clearInterval(interval);
    running = false;
    setStartLabel('Start');
    document.getElementById('pomo-start').classList.remove('active');
}

export function reset() {
    pause();
    isBreak = false;
    seconds = WORK_SECONDS;
    setStartLabel('Start');
    updateDisplay();
}

export function skip() {
    pause();
    switchPhase();
    updateDisplay();
}

function tick() {
    seconds--;
    updateDisplay();
    if (seconds <= 0) {
        pause();
        notify(isBreak ? 'Break over! Back to work.' : 'Time for a break!');
        switchPhase();
        updateDisplay();
    }
}

function switchPhase() {
    isBreak = !isBreak;
    seconds = isBreak ? BREAK_SECONDS : WORK_SECONDS;
    setStartLabel(isBreak ? 'Start Break' : 'Start Work');
}

function updateDisplay() {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    document.getElementById('pomodoro-display').textContent =
        `${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
}

function setStartLabel(label) {
    document.getElementById('pomo-start').textContent = label;
}

function notify(message) {
    if (Notification.permission === 'granted') {
        new Notification(message);
    }
}
