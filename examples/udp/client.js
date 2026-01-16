/**
 * Minimal UDP client using Node.js 'dgram'.
 * Sends a message to a UDP server and prints any response, then exits.
 *
 * Usage:
 *   node client.js "hello world"
 *
 * Environment variables (optional):
 *   HOST    - target host (default: 127.0.0.1)
 *   PORT    - target port (default: 41234)
 *   MESSAGE - message to send (default: "hello from client", overridden by CLI arg)
 *   TIMEOUT - ms to wait for a response before exiting (default: 5000)
 */

const dgram = require('dgram');

const HOST = process.env.HOST || '127.0.0.1';
const PORT = parseInt(process.env.PORT || '41234', 10);
const TIMEOUT_MS = parseInt(process.env.TIMEOUT || '5000', 10);

// Prefer CLI arg over env MESSAGE
const MESSAGE = typeof process.argv[2] === 'string' && process.argv[2].length > 0
  ? process.argv[2]
  : (process.env.MESSAGE || 'hello from client');

const client = dgram.createSocket('udp4');

let timeoutHandle = null;

function exit(code = 0) {
  try { client.close(); } catch {}
  if (timeoutHandle) clearTimeout(timeoutHandle);
  process.exit(code);
}

client.on('error', (err) => {
  console.error('Client error:', err);
  exit(1);
});

client.on('message', (msg, rinfo) => {
  const ts = new Date().toISOString();
  console.log(`[${ts}] Response from ${rinfo.address}:${rinfo.port} -> "${msg.toString('utf8')}"`);
  exit(0);
});

function send() {
  const payload = Buffer.from(MESSAGE, 'utf8');

  client.send(payload, PORT, HOST, (err) => {
    if (err) {
      console.error('Send error:', err);
      exit(1);
      return;
    }
    const ts = new Date().toISOString();
    console.log(`[${ts}] Sent to ${HOST}:${PORT} -> "${MESSAGE}"`);

    // If no response arrives, exit after timeout
    timeoutHandle = setTimeout(() => {
      console.warn(`No response after ${TIMEOUT_MS}ms. Exiting.`);
      exit(0);
    }, TIMEOUT_MS);
  });
}

send();
