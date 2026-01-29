# Synheart CLI
**HSI Mock Data Generator & Broadcaster**

[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/synheart-ai/synheart-cli)](go.mod)

Synheart CLI generates realistic wearable sensor data and transforms it into high-fidelity **HSI (Human State Indicators)** using the integrated **Flux** compute engine (via WebAssembly). It eliminates the dependency on physical devices during HSI-aware application development.

## Architecture

Synheart CLI follows a modern transformation pipeline:
1.  **Sensor Generator**: Produces raw signals (Heart Rate, HRV, Accelerometer, etc.) based on scripted scenarios.
2.  **Vendor Aggregator**: Maps raw signals into vendor-specific payloads (e.g., Whoop or Garmin structures).
3.  **Flux Engine (Optional)**: If `--flux` is provided, transforms vendor payloads into high-fidelity HSI-compliant records.
4.  **Broadcaster**: Streams either the raw vendor JSON or the Flux-generated HSI records over network protocols.

## Features

- 🧠 **Integrated Flux Engine**: Real-time HSI computation powered by the official Synheart Flux Wasm module.
- ⌚ **Vendor-Fidelity**: Support for generating state-of-the-art Whoop or Garmin formatted data.
- 🔄 **Multiple Scenarios**: Baseline, workout, focus session, and more.
- 🌐 **Multi-Transport**: Broadcast HSI over WebSocket, SSE, and UDP.
- CAPTURE **Record & Replay**: Capture Flux-generated HSI sessions for reproducible testing.

## Installation

### Prerequisites

- Go 1.24 or later
- Rust (only if building the Flux Wasm module yourself)

### Build and Install

```bash
git clone https://github.com/synheart-ai/synheart-cli
cd synheart-cli
# This will build the Rust Flux module and the Go CLI
make build
make install
```

## Quick Start

Start the mock server with default settings (streams raw Whoop-formatted data):

```bash
synheart mock start
```

This will:
- Generate sensor data for the `baseline` scenario.
- Aggregate it into Whoop JSON format.
- Start a WebSocket server on `ws://127.0.0.1:8787/hsi`.

To enable **HSI computation** via Flux:
```bash
synheart mock start --flux
```

Connect to the stream from your SDK:

```javascript
const ws = new WebSocket('ws://localhost:8787/hsi');
ws.onmessage = (event) => {
  const hsiRecord = JSON.parse(event.data);
  console.log('New HSI record received:', hsiRecord);
};
```

## Commands

### `synheart mock start`

Start generating and broadcasting real-time sensor data.

```bash
# Basic usage: streams raw Whoop-formatted JSON
synheart mock start

# Enable HSI transformation via Flux
synheart mock start --flux

# Use Garmin data format instead
synheart mock start --vendor garmin

# Debug: See raw vendor JSON before Flux/Broadcast
synheart mock start --flux-verbose
```

**Flags:**
- `--vendor` - Vendor format: `whoop` | `garmin` (default: `whoop`)
- `--flux` - Enable Synheart Flux Wasm transformation to generate HSI
- `--flux-verbose` - Log raw vendor JSON before transformation
- `--scenario` - Scenario to run (default: `baseline`)
- `--duration` - Duration to run (e.g., `5m`, `1h`)
- `--port` - Base port for WebSocket (SSE is port+1, UDP is port+2)

### `synheart mock record`

Record generated HSI records or raw wearable sensor signals to an NDJSON file.

```bash
# Record raw Whoop-formatted signals
synheart mock record --out session.ndjson --vendor whoop

# Record Flux-generated HSI records
synheart mock record --out hsi_session.ndjson --flux
```

### `synheart mock replay`

Replay previously recorded HSI records over network transports with original timing.

```bash
synheart mock replay --in session.ndjson --speed 2.0
```

## Event Schema (HSI 1.0)

Broadcasters emit high-fidelity HSI records computed by Flux:

```json
{
  "hsi_version": "1.0.0",
  "producer": { "name": "flux", "version": "0.1.0", ... },
  "windows": [
    {
      "date": "2024-01-29",
      "physiology": {
        "hrv_rmssd_ms": 50.0,
        "resting_hr_bpm": 60.0,
        "recovery_score": 0.75
      },
      "activity": {
        "strain_score": 0.45,
        "calories": 2200.0
      },
      "sleep": {
        "score": 0.85,
        "efficiency": 0.94
      }
    }
  ]
}
```

## Contributing

Contributions welcome! Please open an issue or PR for new scenarios or vendor formats.

## License

Apache License 2.0. See [`LICENSE`](LICENSE).
