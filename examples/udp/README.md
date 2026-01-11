# UDP Example (Client) — Synheart CLI Integration

This example provides a minimal UDP client for environments where Synheart CLI exposes a UDP transport. It sends a single message to a UDP endpoint and prints any response.

Important note:
- As of the current Synheart CLI README, the primary broadcast transport is WebSocket on `ws://127.0.0.1:8787/hsi`.
- UDP transport is listed in the roadmap. If your build or deployment enables a UDP endpoint, you can use this client to interact with it. Otherwise, use the WebSocket examples for live data.

## Prerequisites

- Node.js 18+ installed
- Synheart CLI cloned or installed (see project README for install and build instructions)

## Files in this folder

- `client.js` — Minimal Node.js UDP client that sends a message and prints any response.

## Running the Synheart CLI server

Refer to the root README for running the server. Typical quick start:

```bash
# Start mock server (WebSocket broadcast)
synheart mock start

# Optional flags
synheart mock start --scenario stress_spike
synheart mock start --port 9000 --duration 5m
```

By default, this starts a WebSocket server at:
- `ws://127.0.0.1:8787/hsi`

If your Synheart CLI build or configuration provides a UDP endpoint, ensure you know:
- UDP HOST (e.g., `127.0.0.1`)
- UDP PORT (e.g., `41234`)

Then configure the UDP client to target that host and port.

## Running the UDP client

From this directory:

```bash
# Basic usage: sends a message and waits for a response (up to TIMEOUT)
node client.js "hello udp"
```

You can also use environment variables:

```bash
# Target a specific UDP host and port; set message and timeout
HOST=127.0.0.1 PORT=41234 MESSAGE="ping from synheart client" TIMEOUT=5000 node client.js
```

Behavior:
- Sends the message via UDP to `HOST:PORT`.
- Prints any response from the server and exits.
- If no response arrives within `TIMEOUT` ms (default 5000), it exits gracefully.

## Matching CLI server settings

If your Synheart CLI exposes UDP:
- Use the same HOST and PORT that the UDP server is bound to.
- For example, if the CLI runs a UDP endpoint at `127.0.0.1:41234`:
  ```bash
  HOST=127.0.0.1 PORT=41234 node client.js "hello from client"
  ```

If UDP is not available:
- Use the WebSocket example (`examples/websocket`) and connect to `ws://localhost:8787/hsi` (or your configured host/port).

## Troubleshooting

- No response: Confirm the UDP endpoint is actually running and reachable. Firewalls may block UDP.
- Address in use: Ensure only one server is bound to the same port.
- Permissions: Binding to ports <1024 may require elevated privileges (server-side). Prefer high-numbered ports for development.

## Notes

- UDP is connectionless and does not guarantee delivery or ordering. If you need reliability, use WebSocket or implement retries/acknowledgements.
- The Synheart CLI README currently emphasizes WebSocket; this UDP client is provided for environments where a UDP transport is enabled, as mentioned in the roadmap.
