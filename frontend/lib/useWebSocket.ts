'use client';

import { useEffect, useRef } from 'react';

export function useWebSocket(
  url: string,
  onMessage: (data: unknown) => void,
  onStatus?: (connected: boolean) => void,
) {
  const onMessageRef = useRef(onMessage);
  const onStatusRef = useRef(onStatus);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const wsRef = useRef<WebSocket | undefined>(undefined);

  useEffect(() => { onMessageRef.current = onMessage; }, [onMessage]);
  useEffect(() => { onStatusRef.current = onStatus; }, [onStatus]);

  useEffect(() => {
    if (typeof window === 'undefined') return;

    function connect() {
      const socket = new WebSocket(url);
      wsRef.current = socket;

      socket.onopen = () => onStatusRef.current?.(true);
      socket.onclose = () => {
        onStatusRef.current?.(false);
        reconnectTimer.current = setTimeout(connect, 3000);
      };
      socket.onerror = () => socket.close();
      socket.onmessage = (e) => {
        try { onMessageRef.current(JSON.parse(e.data as string)); } catch { /* ignore */ }
      };
    }

    connect();

    return () => {
      clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
    };
  }, [url]);
}
