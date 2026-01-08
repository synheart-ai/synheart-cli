# Profiling Synheart CLI

This guide shows how to profile the Synheart CLI using built-in pprof integrations for CPU, memory, blocking, and mutex contention, as well as how to capture an execution trace. You can run profiles against any command; the examples below use `mock start`.

Requirements:
- Go toolchain installed (for `go tool pprof` and `go tool trace`)
- Optional: Graphviz (`dot`) for `pprof -http` visualizations

Contents:
- Quick start
- File-based profiles
- Live pprof HTTP server
- Block and mutex profiling
- Execution traces
- Typical workflows
- Reference of flags and endpoints
- Troubleshooting

---

## Quick start

Profile a short, finite run to keep profiles small and focused.

Example: Run the mock server for 30s and record a CPU profile to a file:
```bash
synheart --cpu-profile cpu.pb.gz mock start --duration 30s
```

Then open the profile:
```bash
go tool pprof -http=:0 cpu.pb.gz
```
This launches a local UI in your browser with flame graphs, call graphs, top tables, and source-level views.

If you prefer live profiling via HTTP endpoints:
```bash
synheart --pprof mock start --duration 60s
```
Then in another terminal:
```bash
# 30-second CPU sample from the running process
go tool pprof -http=:0 http://127.0.0.1:6060/debug/pprof/profile?seconds=30
```

---

## File-based profiles

File-based profiles are convenient for CI or when you want to store artifacts.

- CPU profile
  ```bash
  synheart --cpu-profile cpu.pb.gz mock start --duration 45s
  go tool pprof -http=:0 cpu.pb.gz
  ```
  Notes:
  - CPU profiling starts at process startup and stops on exit.
  - If you don’t specify `--duration`, press Ctrl+C to stop; the profile is flushed on exit.

- Heap (memory) profile
  ```bash
  synheart --mem-profile heap.pb.gz mock start --duration 45s
  go tool pprof -http=:0 heap.pb.gz
  ```
  Notes:
  - The heap profile is captured and written at process exit (after a GC).

- Execution trace
  ```bash
  synheart --trace-profile trace.out mock start --duration 20s
  go tool trace trace.out
  ```
  The trace viewer helps analyze goroutine scheduling, GC pauses, network syscalls, etc.

Tips:
- Use `.pb.gz` for pprof-compatible profiles (common convention).
- Make sure the output directory is writable.

---

## Live pprof HTTP server

Enable the embedded pprof server:
```bash
synheart --pprof --pprof-addr 127.0.0.1:6060 mock start
```

Browse:
- Index: http://127.0.0.1:6060/debug/pprof/
- CPU (30s sample): http://127.0.0.1:6060/debug/pprof/profile?seconds=30
- Heap: http://127.0.0.1:6060/debug/pprof/heap
- Goroutine: http://127.0.0.1:6060/debug/pprof/goroutine
- Block: http://127.0.0.1:6060/debug/pprof/block
- Mutex: http://127.0.0.1:6060/debug/pprof/mutex
- Trace (5s by default): http://127.0.0.1:6060/debug/pprof/trace?seconds=5

Examples:
```bash
# CPU from live server
go tool pprof -http=:0 http://127.0.0.1:6060/debug/pprof/profile?seconds=30

# Heap from live server
go tool pprof -http=:0 http://127.0.0.1:6060/debug/pprof/heap
```

Security:
- By default, the server binds to localhost. Avoid exposing on LAN unless you control access.

---

## Block and mutex profiling

Block and mutex profiles are disabled by default because they add overhead. Enable them only when investigating blocking or lock contention.

- Block profiling:
  ```bash
  synheart --pprof --block-profile-rate 1 mock start --duration 60s
  # Then fetch from the HTTP server:
  go tool pprof -http=:0 http://127.0.0.1:6060/debug/pprof/block
  ```
  The integer rate is the fraction of blocking events sampled. A small non-zero value (1) is a reasonable start.

- Mutex profiling:
  ```bash
  synheart --pprof --mutex-profile-fraction 1 mock start --duration 60s
  go tool pprof -http=:0 http://127.0.0.1:6060/debug/pprof/mutex
  ```

Overhead guidance:
- Start with rate/fraction = 1, increase if profiles are too sparse.
- Turn them off when not in use.

---

## Execution traces

Execution traces are extremely detailed and can be large. Limit duration.

- To file:
  ```bash
  synheart --trace-profile trace.out mock start --duration 10s
  go tool trace trace.out
  ```

- Live (via HTTP):
  ```bash
  synheart --pprof mock start --duration 30s
  # Capture 5 seconds:
  curl -o trace.out "http://127.0.0.1:6060/debug/pprof/trace?seconds=5"
  go tool trace trace.out
  ```

Use cases:
- Analyze goroutine scheduling, GC, syscalls, network and I/O timing.
- Identify stalls in event generation or broadcasting.

---

## Typical workflows

1) Compare performance across changes
```bash
# Baseline
synheart --cpu-profile base.pb.gz mock start --duration 30s

# After changes
synheart --cpu-profile head.pb.gz mock start --duration 30s

# Compare in pprof
go tool pprof base.pb.gz
(pprof) top
(pprof) quit

go tool pprof -base base.pb.gz head.pb.gz
(pprof) top
(pprof) -http=:0   # Visualize differential flame graphs
```

2) Investigate generator hot paths
```bash
synheart --cpu-profile gen.pb.gz mock start --scenario baseline --duration 45s
go tool pprof -http=:0 gen.pb.gz
```
Look for heavy functions under the generator and scenario engine. Consider reducing allocations, improving tick loop efficiency, or batching.

3) Diagnose WebSocket broadcast bottlenecks
```bash
synheart --pprof mock start --duration 60s
# Meanwhile collect CPU and heap:
go tool pprof -http=:0 http://127.0.0.1:6060/debug/pprof/profile?seconds=30
go tool pprof -http=:0 http://127.0.0.1:6060/debug/pprof/heap
```
If contention is suspected:
```bash
synheart --pprof --mutex-profile-fraction 1 --block-profile-rate 1 mock start --duration 60s
go tool pprof -http=:0 http://127.0.0.1:6060/debug/pprof/mutex
go tool pprof -http=:0 http://127.0.0.1:6060/debug/pprof/block
```

4) Capture a short trace for scheduling issues
```bash
synheart --trace-profile trace.out mock start --duration 10s
go tool trace trace.out
```

---

## Flags reference

Global flags relevant to profiling (available to all commands):
- `--pprof` — Enable pprof HTTP server
- `--pprof-addr <host:port>` — Address for pprof server (default: `127.0.0.1:6060`)
- `--cpu-profile <path>` — Write CPU profile to file (starts at process start, stops on exit)
- `--mem-profile <path>` — Write heap profile to file on exit (forces a GC before writing)
- `--trace-profile <path>` — Write execution trace to file
- `--block-profile-rate <n>` — Enable block profiling with given sampling rate (0 to disable)
- `--mutex-profile-fraction <n>` — Enable mutex profiling with given fraction (0 to disable)

Examples:
```bash
# CPU + Heap to files
synheart --cpu-profile cpu.pb.gz --mem-profile heap.pb.gz mock start --duration 30s

# Live pprof server
synheart --pprof --pprof-addr 127.0.0.1:6060 mock start

# Block/Mutex profiling (with live server)
synheart --pprof --block-profile-rate 1 --mutex-profile-fraction 1 mock start --duration 60s
```

---

## HTTP endpoints reference

When `--pprof` is enabled:
- Index: `/debug/pprof/`
- Profiles:
  - `/debug/pprof/profile?seconds=N` — CPU profile (defaults to 30s if omitted)
  - `/debug/pprof/heap` — Heap
  - `/debug/pprof/goroutine`
  - `/debug/pprof/threadcreate`
  - `/debug/pprof/block` — Requires `--block-profile-rate > 0`
  - `/debug/pprof/mutex` — Requires `--mutex-profile-fraction > 0`
  - `/debug/pprof/trace?seconds=N` — Execution trace

---

## Troubleshooting

- The pprof server isn’t reachable
  - Ensure `--pprof` is set and check `--pprof-addr` (default `127.0.0.1:6060`)
  - Confirm nothing else is bound to the port

- CPU profile file is empty or missing
  - Profiles are written on process exit; use `--duration` or Ctrl+C to terminate cleanly
  - Verify the target directory is writable

- Heap profile looks unchanged
  - Heap is captured at exit after a forced GC; ensure the run lasted long enough to allocate
  - Use workload flags (e.g., higher tick rates or longer scenarios) to increase activity

- Visualizations don’t open
  - `pprof -http=:0` requires Graphviz (`dot`) for some views. Install `graphviz`:
    ```bash
    # Ubuntu/Debian
    sudo apt-get install graphviz
    # macOS (Homebrew)
    brew install graphviz
    ```

- High overhead when profiling
  - Disable block/mutex profiles unless needed
  - Keep traces short (5–20s)
  - Avoid profiling everything at once; focus on one dimension (CPU vs Memory vs Contention)

---

## Best practices

- Prefer short, representative runs (30–60s) for CPU profiles.
- Always record a baseline profile before changes and compare with `-base`.
- Keep the pprof server bound to localhost; avoid exposing on untrusted networks.
- Store profile artifacts (`.pb.gz`, `trace.out`) as part of performance regressions investigations and CI runs.

---

Happy profiling!