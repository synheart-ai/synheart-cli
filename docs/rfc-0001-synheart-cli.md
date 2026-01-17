RFC: Synheart CLI — Local HSI Mock Data Generator & Broadcaster

## **1) Summary**

Synheart CLI is a developer tool that **generates HSI-compatible input streams** that normally require **phone + wearables** (e.g., smartwatch sensors). For local development, the CLI **mocks** these sources, produces realistic event payloads, and **broadcasts** them over a selectable or default port so Synheart SDKs can consume data without physical devices.

The CLI is designed to:

- eliminate dependency on watches/phones during development,
- provide repeatable scenarios for QA and demos,
- support multiple SDKs/languages via a stable, documented streaming interface.

## **2) Motivation**

HSI development and debugging currently depends on:

- having a wearable device available,
- pairing / permissions / BLE connectivity,
- inconsistent sensor availability across devices.

This slows iteration and makes onboarding harder. A CLI that generates deterministic, configurable data streams enables:

- fast local SDK testing,
- CI-friendly scenario playback,
- consistent bug reproduction.

## **3) Goals**

- Generate **HSI-compatible** data inputs that mimic phone + wearable sources.
- Broadcast mock streams locally over a **default port** with a simple start command.
- Support **scenarios** (sleep, focus session, stress spike, workout, idle).
- Provide **replay** (recorded session → exact reproduction).
- Provide **deterministic runs** (seeded randomness).
- Keep the tool **language-agnostic** for SDK consumers (Rust/Go/Python/JS/Swift/Kotlin etc.).

## **4) Non-goals**

- Not a full simulator of OS-level BLE stacks or proprietary watch protocols.
- Not a replacement for real device validation.
- Not responsible for model inference; it only produces/broadcasts inputs.
- Not a production ingestion tool (local dev + testing only).

## **5) Terminology**

- **HSI event:** A single time-stamped measurement or derived signal compatible with Synheart HSI ingestion schema.
- **Source:** “phone” or “wearable” (may include subtypes: watch_left, watch_right).
- **Scenario:** A named timeline controlling distributions, spikes, transitions.
- **Transport:** How events are delivered to local SDKs (e.g., WebSocket/UDP/HTTP SSE).

## **6) User stories**

1. **SDK developer:** “I want to run synheart mock start and immediately see my SDK receiving events.”
2. **QA:** “I want a ‘stress spike’ scenario that reproduces the same event pattern every run.”
3. **Product demo:** “I want a 5-minute scripted dataset that looks real without any devices.”
4. **CI:** “I want to run a mock stream headless and validate SDK parsing + local pipeline.”

## **7) Proposed CLI interface**

### **7.1 Commands**

### **synheart mock start**

Starts generating and broadcasting HSI events.

Key flags:

- -transport ws|udp|http-sse (default: ws)
- -host 127.0.0.1 (default)
- -port 8787 (default)
- -scenario <name> (default: baseline)
- -duration 10m (optional; otherwise runs until stopped)
- -rate 50hz (global tick rate; scenario may override per-signal)
- -seed 1234 (deterministic output)
- -device-profile watch_v1|watch_v2|phone_only|custom.json
- -out <file> (record all emitted events to a file)

Example:

- synheart mock start --scenario focus_session --port 8787 --seed 42

### **synheart mock list-scenarios**

Lists built-in scenarios and brief descriptions.

### **synheart mock describe <scenario>**

Outputs scenario config, signals generated, and typical ranges.

### **synheart mock record**

Like start, but emphasizes recording to disk:

- synheart mock record --scenario workout --duration 15m --out workout.ndjson

### **synheart mock replay --in <file>**

Replays a recorded stream exactly (optionally speed-adjusted):

- -speed 1.0 (default)
- -loop
- -transport/host/port same as start

### **synheart doctor**

Checks local environment, port availability, prints connection snippets for SDKs.

### **synheart version**

### **7.2 Exit behavior**

- Graceful shutdown on SIGINT (Ctrl+C): flush buffers, close sockets, finalize recording.

## **8) Data model**

### **8.1 Event envelope (HSI-compatible)**

All generated signals use a consistent envelope (JSON by default):

```
{
  "schema_version": "hsi.input.v1",
  "event_id": "uuid",
  "ts": "2025-12-26T20:05:12.123Z",
  "source": {
    "type": "wearable",
    "id": "mock-watch-01",
    "side": "left"
  },
  "session": {
    "run_id": "uuid",
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

### **8.2 Minimum required signals (v1)**

Initial set should cover the majority of SDK development needs:

**Wearable-like**

- ppg.hr_bpm
- ppg.hrv_rmssd_ms
- accel.xyz_mps2 (vector)
- gyro.xyz_rps (vector)
- temp.skin_c
- eda.us (if supported by the HSI schema)
- spo2.percent (optional)

**Phone-like**

- gps.lat_lon (optional; can be null/disabled)
- screen.state (on/off)
- app.activity (foreground/background or labeled “typing/scrolling” proxies)
- motion.activity (still/walk/run)

Each signal may emit at its own rate (e.g., accel 50Hz, HR 1Hz, HRV 0.2Hz).

### **8.3 File format for record/replay**

- **NDJSON** (one JSON object per line) is the default for streaming friendliness.
- Optional: --format protobuf in future.

## **9) Scenario engine**

### **9.1 Built-in scenarios (initial)**

- baseline: normal day idle + minor variance
- focus_session: reduced motion, stable HR, “screen on” patterns
- stress_spike: sudden HR increase + HRV drop + EDA increase then recovery
- workout: sustained elevated HR, high accel variance
- sleep: low motion, slow HR drift, periodic micro-arousals
- commute: moderate motion + intermittent GPS

### **9.2 Scenario definition**

Scenarios are config-driven (YAML/JSON), compiled into an internal timeline:

- phases (warmup, steady, spike, cooldown),
- per-signal generators (baseline + noise + events),
- correlation rules (e.g., HR ↔ accel intensity, HRV inverse of stress).

Example (conceptual):

```
name: stress_spike
duration: 8m
phases:
  - name: baseline
    for: 2m
  - name: spike
    for: 30s
    overrides:
      ppg.hr_bpm: { add: 35, ramp: 10s }
      ppg.hrv_rmssd_ms: { mul: 0.6 }
      eda.us: { add: 2.0 }
  - name: recovery
    for: 5m30s
```

## **10) Broadcasting transports**

### **10.1 Default: WebSocket**

- URL: ws://127.0.0.1:8787/hsi
- Messages: JSON text frames (1 event per frame) OR batched array (configurable).
- Pros: easy for browsers/Node/mobile, bi-directional (future control channel).

### **10.2 UDP (optional)**

- For high-throughput, low-latency local pipelines.
- Message: one datagram per event or per batch.
- Trade-off: no delivery guarantee.

### **10.3 HTTP SSE (optional)**

- Endpoint: http://127.0.0.1:8787/hsi/stream
- Useful for quick debugging with curl.

## **11) Local control plane (optional but recommended)**

Provide a control endpoint to change behavior without restarting:

- POST /control/pause
- POST /control/resume
- POST /control/scenario (switch scenario)
- POST /control/rate
    
    This is especially useful for demos.
    

## **12) SDK integration expectations**

SDKs should be able to:

- connect to ws://localhost:8787/hsi by default (or configurable),
- parse the event envelope,
- treat "source.type": "wearable" and "phone" as first-class inputs,
- support record/replay for regression tests.

## **13) Security & privacy**

- Runs locally by default bound to 127.0.0.1.
- No real user data; only synthetic.
- If --host 0.0.0.0 is allowed, CLI must print a **warning** about exposure on LAN.
- Recorded files may be shared, but should be labeled “synthetic” in metadata.

## **14) Observability**

- Console logs: scenario, rates, connected clients, drop counts.
- -verbose for per-signal statistics.
- synheart doctor prints:
    - chosen transport endpoints,
    - sample subscription code snippets,
    - port conflict detection.

## **15) Implementation plan**

### **Phase 1 (MVP)**

- WebSocket transport
- baseline + focus_session + stress_spike scenarios
- seed, duration, rate
- NDJSON record/replay
- doctor command

### **Phase 2**

- workout + sleep + commute
- HTTP SSE + UDP
- control plane endpoints

### **Phase 3**

- pluggable custom scenarios (load from file/dir)
- protobuf option
- “schema validation mode” (fail fast if payload deviates)


## **16) Appendix: Example usage**

- Fast start:
    - synheart mock start
- Stress spike demo:
    - synheart mock start --scenario stress_spike --seed 7 --duration 5m
- Record then replay:
    - synheart mock record --scenario workout --duration 12m --out workout.ndjson
    - synheart mock replay --in workout.ndjson --speed 1.25

---