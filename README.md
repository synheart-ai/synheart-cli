# Synheart CLI

**Local HSI Mock Data Generator & Broadcaster for SDK Development**

[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/synheart-ai/synheart-cli)](go.mod)
[![CI](https://github.com/synheart-ai/synheart-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/synheart-ai/synheart-cli/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/synheart-ai/synheart-cli)](https://github.com/synheart-ai/synheart-cli/releases)

Synheart CLI generates HSI-compatible sensor data streams that mimic phone + wearable sources, eliminating dependency on physical devices during development.

Changelog: see [`CHANGELOG.md`](CHANGELOG.md).

## Features

- 🎯 **Mock HSI Data Streams**: Generate realistic sensor data without physical devices
- 🔄 **Multiple Scenarios**: Baseline, focus session, stress spike, and more
- 🌐 **WebSocket Broadcasting**: Real-time data streaming on localhost
- 📼 **Record & Replay**: Capture sessions for reproducible testing
- 🎲 **Deterministic Mode**: Use seeds for consistent, repeatable data generation
- 🛠️ **Developer Friendly**: Simple CLI, clear documentation, easy SDK integration

## Installation

### Prerequisites

- Go 1.24 or later

### Install Globally (Recommended)

Install the CLI to make it available system-wide:

```bash
git clone https://github.com/synheart-ai/synheart-cli
cd synheart-cli
make install
```

This installs the binary to `$GOPATH/bin` (typically `$HOME/go/bin`).

**Ensure Go's bin directory is in your PATH:**

Add this line to your `~/.zshrc`, `~/.bashrc`, or `~/.bash_profile`:

```bash
export PATH="$PATH:$HOME/go/bin"
```

Then reload your shell configuration:

```bash
source ~/.zshrc  # or ~/.bashrc
```

**Verify installation:**

```bash
synheart version
```

### Shell Completion (Recommended)

Generate and load completions for your shell:

```bash
# Zsh
mkdir -p ~/.zsh/completions
synheart completion zsh > ~/.zsh/completions/_synheart
```

```bash
# Bash
mkdir -p ~/.bash_completion.d
synheart completion bash > ~/.bash_completion.d/synheart
```

Or let `make install` handle it:

```bash
make install INSTALL_COMPLETIONS=zsh
```

### Build Locally (Development)

To build without installing globally:

```bash
make build
```

To build with release metadata (injects version + git commit into `synheart version`):

```bash
make build VERSION=0.0.1
```

The binary will be available at `bin/synheart`. Run it with:

```bash
./bin/synheart version
```

## Global Flags

These flags apply to most commands:

- `--format text|json` (default: `text`) — JSON is supported by `version`, `doctor`, `mock list-scenarios`, and `mock describe`
- `--no-color` — Disable colored output
- `-q, --quiet` — Suppress non-essential output
- `-v, --verbose` — Verbose logging

## Quick Start

Start the mock server with default settings:

```bash
synheart mock start
```

This will:
- Generate baseline scenario data
- Start WebSocket server on `ws://127.0.0.1:8787/hsi`
- Broadcast events to all connected clients

Connect to the stream from your SDK:

```javascript
const ws = new WebSocket('ws://localhost:8787/hsi');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(data);
};
```

## Commands

### `synheart mock start`

Start generating and broadcasting HSI events.

```bash
# Basic usage
synheart mock start

# With specific scenario
synheart mock start --scenario stress_spike

# Deterministic output with seed
synheart mock start --scenario focus_session --seed 42

# Custom port and duration
synheart mock start --port 9000 --duration 5m

# Record to file while broadcasting
synheart mock start --scenario workout --out workout.ndjson
```

**Flags:**
- `--host` - Host to bind to (default: `127.0.0.1`)
- `--port` - Port to listen on (default: `8787`)
- `--scenario` - Scenario to run (default: `baseline`)
- `--duration` - Duration to run (e.g., `5m`, `1h`)
- `--rate` - Global tick rate (default: `50hz`)
- `--seed` - Random seed for deterministic output
- `--out` - Record events to file (NDJSON format)

### `synheart mock record`

Record mock data to an NDJSON file.

```bash
synheart mock record --scenario workout --duration 15m --out workout.ndjson
```

**Flags:**
- `--scenario` - Scenario to run (default: `baseline`)
- `--duration` - Duration to record (default: `5m`)
- `--out` - Output file (required)
- `--seed` - Random seed for deterministic output
- `--rate` - Global tick rate (default: `50hz`)

### `synheart mock replay`

Replay events from a recorded file.

```bash
# Basic replay
synheart mock replay --in workout.ndjson

# Faster playback
synheart mock replay --in workout.ndjson --speed 2.0

# Loop continuously
synheart mock replay --in workout.ndjson --loop
```

**Flags:**
- `--in` - Input file to replay (required)
- `--speed` - Playback speed multiplier (default: `1.0`)
- `--loop` - Loop playback continuously
- `--host` - Host to bind to (default: `127.0.0.1`)
- `--port` - Port to listen on (default: `8787`)

### `synheart mock list-scenarios`

List all available scenarios.

```bash
synheart mock list-scenarios
```

### `synheart mock describe <scenario>`

Show detailed information about a scenario.

```bash
synheart mock describe stress_spike
```

### `synheart doctor`

Check environment and print connection examples.

```bash
synheart doctor
```

Validates:
- Scenarios directory exists
- Default port availability
- Provides SDK connection examples

### `synheart version`

Print version information.

```bash
synheart version
```

## Built-in Scenarios

### `baseline`
Normal day idle with minor variance. Suitable for testing basic SDK functionality.

### `focus_session`
Reduced motion, stable heart rate, screen on patterns. Simulates a 30-minute focused work session.

### `stress_spike`
Sudden heart rate increase, HRV drop, EDA spike followed by recovery. 8-minute scenario for testing stress detection.

## Event Schema

All events follow the HSI-compatible envelope format:

```json
{
  "schema_version": "hsi.input.v1",
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "ts": "2025-12-26T20:05:12.123Z",
  "source": {
    "type": "wearable",
    "id": "mock-watch-01",
    "side": "left"
  },
  "session": {
    "run_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "scenario": "focus_session",
    "seed": 42
  },
  "signal": {
    "name": "ppg.hr_bpm",
    "unit": "bpm",
    "value": 72.4,
    "quality": 0.93
  },
  "meta": {
    "sequence": 12093
  }
}
```

## Supported Signals

### Wearable Signals
- `ppg.hr_bpm` - Heart rate (beats per minute)
- `ppg.hrv_rmssd_ms` - Heart rate variability (milliseconds)
- `accel.xyz_mps2` - 3D acceleration (m/s²)
- `gyro.xyz_rps` - 3D gyroscope (rad/s)
- `temp.skin_c` - Skin temperature (Celsius)
- `eda.us` - Electrodermal activity (microsiemens)

### Phone Signals
- `screen.state` - Screen on/off state
- `app.activity` - App foreground/background activity
- `motion.activity` - Motion activity (still/walk/run)

## SDK Integration Examples

### JavaScript/Node.js

```javascript
const WebSocket = require('ws');

const ws = new WebSocket('ws://localhost:8787/hsi');

ws.on('message', (data) => {
  const event = JSON.parse(data);
  console.log(`${event.signal.name}: ${event.signal.value}`);
});

ws.on('error', (error) => {
  console.error('WebSocket error:', error);
});
```

### Python

```python
import websocket
import json

def on_message(ws, message):
    event = json.loads(message)
    print(f"{event['signal']['name']}: {event['signal']['value']}")

def on_error(ws, error):
    print(f"Error: {error}")

ws = websocket.WebSocketApp(
    "ws://localhost:8787/hsi",
    on_message=on_message,
    on_error=on_error
)

ws.run_forever()
```

### Go

```go
package main

import (
	"encoding/json"
	"log"
	"github.com/gorilla/websocket"
)

type Event struct {
	Signal struct {
		Name  string      `json:"name"`
		Value interface{} `json:"value"`
	} `json:"signal"`
}

func main() {
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8787/hsi", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Fatal(err)
		}

		var event Event
		json.Unmarshal(message, &event)
		log.Printf("%s: %v", event.Signal.Name, event.Signal.Value)
	}
}
```

### Rust

```rust
use tokio_tungstenite::{connect_async, tungstenite::Message};
use futures_util::StreamExt;
use serde_json::Value;

#[tokio::main]
async fn main() {
    let (mut socket, _) = connect_async("ws://localhost:8787/hsi")
        .await
        .expect("Failed to connect");

    while let Some(msg) = socket.next().await {
        if let Ok(Message::Text(text)) = msg {
            let event: Value = serde_json::from_str(&text).unwrap();
            println!("{}: {}",
                event["signal"]["name"].as_str().unwrap(),
                event["signal"]["value"]
            );
        }
    }
}
```

## Development

### Building

```bash
make build
```

### Running Tests

```bash
make test
```

### Cleaning

```bash
make clean
```

## Use Cases

- **SDK Development**: Test HSI SDKs without physical wearables
- **CI/CD**: Automated testing with deterministic data
- **Demos**: Scripted scenarios that look realistic
- **QA**: Reproducible test cases with seeded data
- **Integration Testing**: Validate data pipelines locally

## Recording Format

Recordings use NDJSON (Newline Delimited JSON) format - one JSON object per line. This format is:
- Streaming-friendly
- Easy to parse
- Human-readable
- Compatible with standard tools

Example:

```
{"schema_version":"hsi.input.v1","event_id":"...","ts":"2025-12-26T20:05:12.123Z",...}
{"schema_version":"hsi.input.v1","event_id":"...","ts":"2025-12-26T20:05:12.143Z",...}
{"schema_version":"hsi.input.v1","event_id":"...","ts":"2025-12-26T20:05:12.163Z",...}
```

## Security Notes

- Binds to `localhost` (127.0.0.1) by default
- Only generates synthetic data, never real user data
- Use `--host 0.0.0.0` with caution (exposes on LAN)

## Troubleshooting

### Port already in use

If port 8787 is already in use:

```bash
synheart mock start --port 9000
```

### Scenarios not found

Ensure the `scenarios/` directory is in:
1. Current working directory
2. Same directory as the executable

Use `synheart doctor` to check.

### WebSocket connection refused

Ensure the server is running and check firewall settings.

## Contributing

Contributions welcome! Please open an issue or PR.

## Roadmap

See RFC document for planned Phase 2 and Phase 3 features:
- Additional scenarios (workout, sleep, commute)
- HTTP SSE and UDP transports
- Control plane for runtime scenario switching
- Custom scenario loading
- Protobuf support

## License

Apache License 2.0. See [`LICENSE`](LICENSE).

## Patent Pending Notice

This project is provided under an open-source license. Certain underlying systems, methods, and architectures described or implemented herein may be covered by one or more pending patent applications.

Nothing in this repository grants any license, express or implied, to any patents or patent applications, except as provided by the applicable open-source license.

