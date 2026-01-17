## RFC-0002: Synheart CLI - Receiver Mode for HSI Data Ingestion

---

**Applies to:** Synheart CLI

**Related Systems:** Synheart Life App, Synheart Core, HSI Export Schema


---

## **1. Overview**

This RFC proposes adding a **Receiver Mode** to the Synheart CLI that allows users to:

- Run a lightweight local or server-side HTTP endpoint
- Receive **HSI export JSON** directly from the Synheart Life app
- Store, forward, or process the data locally
- Maintain full user ownership of data without relying on Synheart cloud services

This feature enables **peer-to-peer data delivery** from mobile to PC/server and reinforces Synheart’s commitment to portability, transparency, and open infrastructure.

---

## **2. Motivation**

Users have expressed the need to:

- Export their data in structured formats (JSON)
- Send HSI summaries directly to their own machines
- Integrate Synheart data into personal workflows, analytics pipelines, or research environments

Relying solely on file export or Synheart cloud creates unnecessary friction.

A local receiver provides:

- Faster iteration
- No third-party dependency
- Maximum privacy
- Developer and researcher friendliness

---

## **3. Design Principles**

1. **User-initiated only**
    
    The receiver only accepts data explicitly sent by the user.
    
2. **Derived data only**
    
    Only HSI summaries and insights are supported.
    
3. **Simple by default**
    
    One command to start, one endpoint to receive.
    
4. **Secure but lightweight**
    
    Token-based auth, no complex PKI in v1.
    
5. **Open & extensible**
    
    Receiver is open-source and scriptable.
    

---

## **4. Feature Summary**

The Synheart CLI will support a new mode:

```
synheart receiver
```

This command starts a local HTTP server that listens for incoming **HSI export payloads** from the Synheart Life app.

---

## **5. CLI Interface**

### **5.1 Basic Usage**

```
synheart receiver
```

Defaults:

- Host: 0.0.0.0
- Port: 8787
- Output: stdout (pretty-printed JSON)
- Auth: auto-generated bearer token

---

### **5.2 Advanced Options**

```
synheart receiver \
  --port 8787 \
  --host 127.0.0.1 \
  --token <custom-token> \
  --out ./exports \
  --format json \
  --gzip
```

| **Flag** | **Description** |
| --- | --- |
| --port | Port to listen on |
| --host | Bind address |
| --token | Static bearer token (optional) |
| --out | Directory to write received payloads |
| --format | json or ndjson |
| --gzip | Accept gzip-compressed payloads |

---

## **6. Receiver Endpoint Contract**

### **6.1 HTTP Endpoint**

```
POST /v1/hsi/import
```

### **6.2 Required Headers**

```
Content-Type: application/json
Authorization: Bearer <token>
X-Synheart-Schema: synheart.hsi.export.v1
X-Synheart-Export-Id: <uuid>
X-Synheart-Sent-At: <utc>
Idempotency-Key: <uuid>
```

### **6.3 Request Body**

The payload must conform to the **HSI Export Schema v1** as defined in the Synheart Life Data Export RFC.

---

## **7. Authentication & Security**

### **7.1 Token Generation**

- If no token is provided, the CLI generates a random token on startup.
- Token is printed once to stdout.

Example:

```
Receiver started
Endpoint: http://192.168.1.10:8787/v1/hsi/import
Auth token: sh_9f83a1c...
```

### **7.2 Transport Security**

- HTTPS is recommended for remote servers
- HTTP is allowed for:
    - localhost
    - LAN usage

Future versions may support:

- mTLS
- QR-based pairing

---

## **8. Data Handling Behavior**

Upon receiving a valid payload:

1. Validate headers and schema
2. Validate JSON structure
3. Check idempotency key
4. Write payload to disk (if --out is set)
5. Emit a summary to stdout

Example stdout:

```
✔ Received export
  ID: 7c2a…
  Range: 2025-12-16 → 2026-01-16
  Summaries: 4320
  Insights: 6
```

---

## **9. Failure Handling**

| **Scenario** | **Behavior** |
| --- | --- |
| Invalid token | 401 Unauthorized |
| Invalid schema | 400 Bad Request |
| Duplicate export_id | 200 OK (ignored) |
| Disk write failure | 500 + error log |
| Client disconnect | Partial payload discarded |

No retries are initiated by the receiver; retries are the sender’s responsibility.

---

## **10. Output & Integration**

### **10.1 File Output**

If --out is specified:

- One file per export
- Filename format:

```
synheart_export_<export_id>.json
```

### **10.2 Streaming Mode (Future)**

Support piping to other tools:

```
synheart receiver | jq '.summaries[]'
```

---

## **11. Non-Goals (v1)**

- No background syncing
- No raw keystroke ingestion
- No automatic cloud forwarding
- No UI dashboard
- No multi-user auth

---

## **12. Open Questions**

1. Should we support WebSocket ingestion later? ( YES)
2. Should the receiver auto-rotate tokens? ( YES)
3. Should the CLI provide a QR pairing helper? (YES)
4. Should we add optional validation against HSI 1.0 schemas? (YES)

---

## **13. Rollout Plan**

### **Phase 1**

- Implement receiver mode
- JSON validation
- Token auth
- File output

### **Phase 2**

- QR pairing
- NDJSON streaming
- Optional TLS helper
- Plugin hooks

---

## **14. Summary**

Adding a **Receiver Mode** to Synheart CLI:

- Strengthens user data ownership
- Complements local-first design
- Enables research and advanced workflows
- Avoids unnecessary cloud dependency

This feature positions Synheart CLI as a **first-class bridge** between mobile human-state data and user-controlled systems.

---

## **15. One-Line Principle (Internal)**

> If users own their state, they must control where it goes.
> 

---