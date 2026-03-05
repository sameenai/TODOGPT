// ws.js — WebSocket connection with automatic reconnect.
// Calls onMessage(update) for each parsed update, onStatus(connected) on state changes.

export function connect(onMessage, onStatus) {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const socket = new WebSocket(`${proto}//${location.host}/ws`);

    socket.onopen = () => onStatus(true);

    socket.onclose = () => {
        onStatus(false);
        setTimeout(() => connect(onMessage, onStatus), 3000);
    };

    socket.onmessage = (event) => {
        try {
            onMessage(JSON.parse(event.data));
        } catch (e) {
            console.error('WebSocket parse error:', e);
        }
    };
}
