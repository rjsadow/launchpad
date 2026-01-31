import { useEffect, useRef, useCallback, useImperativeHandle, forwardRef } from 'react';
import type RFB from '@novnc/novnc/lib/rfb.js';

// Clipboard policy: 'allow' enables bidirectional clipboard, 'deny' disables it
export type ClipboardPolicy = 'allow' | 'deny';

interface VNCViewerProps {
  wsUrl: string;
  onConnect?: () => void;
  onDisconnect?: (clean: boolean) => void;
  onError?: (message: string) => void;
  viewOnly?: boolean;
  scaleViewport?: boolean;
  clipboardPolicy?: ClipboardPolicy;
  autoReconnect?: boolean;
  maxReconnectAttempts?: number;
}

export interface VNCViewerHandle {
  disconnect: () => void;
  reconnect: () => void;
  requestFullscreen: () => Promise<void>;
}

export const VNCViewer = forwardRef<VNCViewerHandle, VNCViewerProps>(function VNCViewer({
  wsUrl,
  onConnect,
  onDisconnect,
  onError,
  viewOnly = false,
  scaleViewport = true,
  clipboardPolicy = 'deny',
  autoReconnect = true,
  maxReconnectAttempts = 3,
}, ref) {
  const containerRef = useRef<HTMLDivElement>(null);
  const rfbRef = useRef<RFB | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const isConnectedRef = useRef(false);
  const wsUrlRef = useRef(wsUrl);

  // Keep wsUrl ref updated
  useEffect(() => {
    wsUrlRef.current = wsUrl;
  }, [wsUrl]);

  const cleanupRFB = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    if (rfbRef.current) {
      try {
        rfbRef.current.disconnect();
      } catch {
        // Ignore disconnect errors
      }
      rfbRef.current = null;
    }
  }, []);

  const initRFB = useCallback(async () => {
    if (!containerRef.current || !wsUrlRef.current) {
      return;
    }

    // Build full WebSocket URL
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const fullWsUrl = wsUrlRef.current.startsWith('ws')
      ? wsUrlRef.current
      : `${protocol}//${window.location.host}${wsUrlRef.current}`;

    try {
      // Dynamic import
      const { default: RFBClass } = await import('@novnc/novnc/lib/rfb.js');

      if (!containerRef.current) return;

      // Create RFB connection
      const rfb = new RFBClass(containerRef.current, fullWsUrl, {
        shared: true,
      });

      // Configure RFB
      rfb.scaleViewport = scaleViewport;
      rfb.resizeSession = false;
      rfb.viewOnly = viewOnly;
      rfb.clipViewport = false;

      // Clipboard policy: noVNC doesn't have a direct enable/disable clipboard option,
      // but we can control it by not calling clipboardPasteFrom when policy is 'deny'
      // The actual clipboard events are handled by the browser and noVNC internally

      // Add event listeners
      rfb.addEventListener('connect', () => {
        console.log('VNC connected');
        isConnectedRef.current = true;
        reconnectAttemptsRef.current = 0;
        onConnect?.();
      });

      rfb.addEventListener('disconnect', (e: unknown) => {
        const event = e as { detail?: { clean?: boolean } };
        const clean = event?.detail?.clean ?? false;
        console.log('VNC disconnected, clean:', clean);
        isConnectedRef.current = false;

        // Attempt reconnect if disconnection was not clean and autoReconnect is enabled
        if (!clean && autoReconnect && reconnectAttemptsRef.current < maxReconnectAttempts) {
          reconnectAttemptsRef.current++;
          const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current - 1), 10000);
          console.log(`VNC reconnect attempt ${reconnectAttemptsRef.current}/${maxReconnectAttempts} in ${delay}ms`);

          reconnectTimeoutRef.current = window.setTimeout(() => {
            cleanupRFB();
            initRFB();
          }, delay);
        } else {
          onDisconnect?.(clean);
        }
      });

      rfb.addEventListener('securityfailure', (e: unknown) => {
        const event = e as { detail?: { reason?: string } };
        const reason = event?.detail?.reason ?? 'Unknown security failure';
        console.error('VNC security failure:', reason);
        onError?.(reason);
      });

      // Handle clipboard based on policy
      if (clipboardPolicy === 'allow') {
        rfb.addEventListener('clipboard', (e: unknown) => {
          const event = e as { detail?: { text?: string } };
          const text = event?.detail?.text;
          if (text && navigator.clipboard) {
            navigator.clipboard.writeText(text).catch((err) => {
              console.warn('Failed to write to clipboard:', err);
            });
          }
        });
      }

      rfbRef.current = rfb;
    } catch (err) {
      console.error('Failed to load or initialize noVNC:', err);
      onError?.(err instanceof Error ? err.message : 'Failed to load VNC viewer');
    }
  }, [scaleViewport, viewOnly, clipboardPolicy, autoReconnect, maxReconnectAttempts, onConnect, onDisconnect, onError, cleanupRFB]);

  // Expose methods via ref
  useImperativeHandle(ref, () => ({
    disconnect: () => {
      reconnectAttemptsRef.current = maxReconnectAttempts; // Prevent auto-reconnect
      cleanupRFB();
    },
    reconnect: () => {
      reconnectAttemptsRef.current = 0;
      cleanupRFB();
      initRFB();
    },
    requestFullscreen: async () => {
      if (containerRef.current) {
        try {
          await containerRef.current.requestFullscreen();
        } catch (err) {
          console.error('Failed to enter fullscreen:', err);
          throw err;
        }
      }
    },
  }), [cleanupRFB, initRFB, maxReconnectAttempts]);

  useEffect(() => {
    initRFB();

    // Cleanup on unmount
    return () => {
      cleanupRFB();
    };
  }, [initRFB, cleanupRFB]);

  return (
    <div
      ref={containerRef}
      className="w-full h-full bg-black"
      style={{ minHeight: '400px' }}
    />
  );
});
