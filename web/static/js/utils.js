// utils.js — pure helpers with no side effects and no DOM access

export function escapeHtml(s) {
    if (!s) return '';
    const div = document.createElement('div');
    div.appendChild(document.createTextNode(s));
    return div.innerHTML;
}

export function timeAgo(dateStr) {
    if (!dateStr) return '';
    const diff = Math.floor((Date.now() - new Date(dateStr)) / 1000);
    if (diff < 60)    return 'just now';
    if (diff < 3600)  return Math.floor(diff / 60) + 'm ago';
    if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
    return Math.floor(diff / 86400) + 'd ago';
}

export function formatTime(dateStr) {
    return new Date(dateStr).toLocaleTimeString('en-US', {
        hour: 'numeric', minute: '2-digit', hour12: true,
    });
}

export function formatDate(dateStr) {
    return new Date(dateStr).toLocaleDateString('en-US', {
        month: 'short', day: 'numeric',
    });
}

export function capitalize(s) {
    if (!s) return '';
    return s.charAt(0).toUpperCase() + s.slice(1);
}
