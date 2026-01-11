# WebSocket Example

This example provides:
- A minimal WebSocket server using Node.js and the `ws` library.
- A minimal HTML client that connects to the server and exchanges messages.

The WebSocket protocol enables full-duplex communication between client and server over a single, long-lived TCP connection.

## Prerequisites

- Node.js 18+ installed
- A terminal
- Optional: a static file server to serve the HTML client (or open the HTML file directly in the browser)

## Folder Contents

- `server.js` — Node.js WebSocket server (using `ws`)
- `client.html` — Minimal HTML/JS WebSocket client

## Server: `server.js`

```js
/**
 * Minimal WebSocket server using 'ws'.
 *
 * - Listens on HOST:PORT (default 0.0.0.0:8081)
 * - Logs connections and messages
 * - Echoes incoming messages back to the sender
 * - Broadcasts a periodic message to all connected clients
 *
 * Usage:
 *   npm i ws
 *   HOST=0.0.0.0 PORT=8081 node server.js
 */

import http from 'node:http';
import { WebSocketServer } from 'ws';

const HOST = process.env.HOST || '0.0.0.0';
const PORT = parseInt(process.env.PORT || '8081', 10);

const server = http.createServer();
const wss = new WebSocketServer({ server });

function broadcast(data) {
  const payload = typeof data === 'string' ? data : JSON.stringify(data);
  for (const client of wss.clients) {
    if (client.readyState === 1) { // WebSocket.OPEN
      client.send(payload);
    }
  }
}

wss.on('connection', (ws, req) => {
  const ip = req.socket.remoteAddress;
  console.log(`[${new Date().toISOString()}] Client connected: ${ip}`);

  ws.on('message', (message) => {
    const text = message.toString('utf8');
    console.log(`[${new Date().toISOString()}] Received from ${ip}: ${text}`);

    // Echo back to sender
    ws.send(JSON.stringify({ type: 'echo', payload: text }));

    // Optionally broadcast to others
    // broadcast({ type: 'broadcast', from: ip, payload: text });
  });

  ws.on('close', () => {
    console.log(`[${new Date().toISOString()}] Client disconnected: ${ip}`);
  });

  ws.on('error', (err) => {
    console.error('WebSocket error:', err);
  });

  // Send a welcome message
  ws.send(JSON.stringify({ type: 'welcome', message: 'Connected to WebSocket server!' }));
});

// Periodic broadcast example
setInterval(() => {
  broadcast({ type: 'tick', ts: new Date().toISOString() });
}, 3000);

server.listen(PORT, HOST, () => {
  console.log(`WebSocket server listening on ws://${HOST}:${PORT}`);
});
```

## Client: `client.html`

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>Minimal WebSocket Client</title>
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>
    body { margin: 0; font-family: system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial, sans-serif; background: #0f172a; color: #e2e8f0; }
    .container { max-width: 800px; margin: 0 auto; padding: 16px; }
    h1 { font-size: 18px; margin: 0 0 12px; font-weight: 600; }
    .controls { display: grid; grid-template-columns: 1fr auto auto auto; gap: 8px; margin-bottom: 12px; }
    input, button { padding: 8px 10px; border-radius: 8px; border: 1px solid #334155; background: #0b1220; color: #e2e8f0; }
    button { cursor: pointer; }
    button:hover { filter: brightness(1.08); }
    .status { display: inline-flex; align-items: center; gap: 8px; font-size: 13px; padding: 6px 10px; border-radius: 999px; border: 1px solid #334155; background: #0b1220; margin: 8px 0 12px; }
    .dot { width: 10px; height: 10px; border-radius: 50%; background: #ef4444; }
    .dot.connected { background: #22c55e; }
    .dot.connecting { background: #f59e0b; }
    .card { border: 1px solid #334155; border-radius: 12px; background: #0b1220; overflow: hidden; margin-top: 12px; }
    .card h2 { font-size: 15px; margin: 0; padding: 10px 12px; border-bottom: 1px solid #334155; background: #0e1626; }
    .card-body { padding: 12px; max-height: 420px; overflow: auto; }
    .log { border: 1px solid #334155; border-radius: 8px; padding: 10px; margin-bottom: 10px; background: #0f1a2c; }
    pre { margin: 0; white-space: pre-wrap; word-wrap: break-word; font-size: 13px; line-height: 1.35; color: #cbd5e1; }
    .send { display: grid; grid-template-columns: 1fr auto; gap: 8px; margin-top: 12px; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Minimal WebSocket Client</h1>

    <div class="status" id="status">
      <span class="dot" id="statusDot"></span>
      <span id="statusText">Disconnected</span>
    </div>

    <div class="controls">
      <input id="endpoint" type="text" placeholder="ws://localhost:8081" />
      <button id="connectBtn">Connect</button>
      <button id="disconnectBtn" disabled>Disconnect</button>
      <button id="clearBtn">Clear</button>
    </div>

    <div class="card">
      <h2>Messages</h2>
      <div class="card-body" id="logs"></div>
    </div>

    <div class="send">
      <input id="outgoing" type="text" placeholder='Type a message (e.g., {"cmd":"ping"})' />
      <button id="sendBtn" disabled>Send</button>
    </div>
  </div>

  <script>
    (function () {
      const $ = (id) => document.getElementById(id);

      const endpointInput = $('endpoint');
      const connectBtn = $('connectBtn');
      const disconnectBtn = $('disconnectBtn');
      const clearBtn = $('clearBtn');
      const logsEl = $('logs');
      const statusDot = $('statusDot');
      const statusText = $('statusText');
      const outgoingInput = $('outgoing');
      const sendBtn = $('sendBtn');

      let ws = null;

      function setStatus(state, text) {
        statusDot.classList.remove('connected', 'connecting');
        if (state === 'connected') statusDot.classList.add('connected');
        else if (state === 'connecting') statusDot.classList.add('connecting');
        statusText.textContent = text;
      }

      function log(kind, payload) {
        const wrapper = document.createElement('div');
        wrapper.className = 'log';
        const pre = document.createElement('pre');
        const ts = new Date().toLocaleTimeString();
        let text = payload;
        try {
          const parsed = typeof payload === 'string' ? JSON.parse(payload) : payload;
          text = JSON.stringify(parsed, null, 2);
        } catch { /* keep as string */ }
        pre.textContent = `[${ts}] ${kind}\n` + (typeof text === 'string' ? text : String(text));
        wrapper.appendChild(pre);
        logsEl.appendChild(wrapper);
        logsEl.scrollTop = logsEl.scrollHeight;
      }

      function clearLogs() {
        logsEl.innerHTML = '';
      }

      function connect() {
        const url = (endpointInput.value || '').trim();
        if (!url) {
          setStatus('disconnected', 'Please enter a valid ws:// or wss:// URL');
          return;
        }

        disconnect(false);
        setStatus('connecting', 'Connecting…');

        try {
          ws = new WebSocket(url);
        } catch {
          setStatus('disconnected', 'Invalid URL');
          return;
        }

        ws.onopen = () => {
          setStatus('connected', 'Connected');
          connectBtn.disabled = true;
          disconnectBtn.disabled = false;
          sendBtn.disabled = false;
          log('open', `Connected to ${url}`);
        };

        ws.onmessage = (ev) => {
          log('message', ev.data);
        };

        ws.onerror = (ev) => {
          log('error', 'WebSocket error');
        };

        ws.onclose = () => {
          setStatus('disconnected', 'Disconnected');
          connectBtn.disabled = false;
          disconnectBtn.disabled = true;
          sendBtn.disabled = true;
          log('close', 'Connection closed');
        };
      }

      function disconnect(updateStatus = true) {
        if (ws) {
          try { ws.close(); } catch {}
          ws = null;
        }
        if (updateStatus) {
          setStatus('disconnected', 'Disconnected');
          connectBtn.disabled = false;
          disconnectBtn.disabled = true;
          sendBtn.disabled = true;
        }
      }

      function send() {
        const payload = (outgoingInput.value || '').trim();
        if (!payload) return;
        if (!ws || ws.readyState !== WebSocket.OPEN) {
          log('warn', 'WebSocket is not open');
          return;
        }
        ws.send(payload);
        log('sent', payload);
        outgoingInput.value = '';
      }

      // Wire up UI
      connectBtn.addEventListener('click', connect);
      disconnectBtn.addEventListener('click', () => disconnect(true));
      clearBtn.addEventListener('click', clearLogs);
      sendBtn.addEventListener('click', send);
      outgoingInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') send();
      });

      // Persist endpoint between sessions
      const STORAGE_KEY = 'ws_endpoint';
      endpointInput.value = localStorage.getItem(STORAGE_KEY) || 'ws://localhost:8081';
      endpointInput.addEventListener('change', () => {
        localStorage.setItem(STORAGE_KEY, endpointInput.value);
      });
    })();
  </script>
</body>
</html>
```

## How to Run

1. Install server dependency:
   ```
   npm i ws
   ```

2. Start the WebSocket server from this directory:
   ```
   node server.js
   ```
   Optional environment overrides:
   ```
   HOST=0.0.0.0 PORT=8081 node server.js
   ```

3. Open the client:
   - Option A: Open `client.html` directly in your browser and set the endpoint to `ws://localhost:8081`.
   - Option B: Serve the directory with a static server (for same-origin), then open `client.html`:
     ```
     npx http-server . -p 8080
     ```
     Then visit `http://localhost:8080/synheart-cli/examples/websocket/client.html` and connect to `ws://localhost:8081`.

4. Send messages from the client using the input box and observe echoed responses and periodic `tick` broadcasts.

## Tips

- Use `wss://` when serving over HTTPS to avoid mixed-content issues.
- If you need authentication, consider a cookie-based session or perform an HTTP auth handshake before upgrading to WebSocket, or pass a token via query string and validate it server-side.
- For production, handle heartbeats/pings and reconnect logic on the client to improve resilience.

## Troubleshooting

- If the client can’t connect, confirm the server is listening and the URL is correct.
- Check firewall settings that might block the server port.
- When reverse proxying (e.g., via Nginx), ensure WebSocket upgrade headers are properly forwarded:
  ```
  proxy_set_header Upgrade $http_upgrade;
  proxy_set_header Connection "upgrade";
  ```
