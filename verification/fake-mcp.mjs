#!/usr/bin/env node
// Minimal stdio MCP server (newline-delimited JSON-RPC) exposing one tool.
// Used by probe.mjs so mcp_status shows a "connected" server with tools,
// and by the paid scenarios as a deterministic, side-effect-free tool target.
import { createInterface } from "node:readline";
const rl = createInterface({ input: process.stdin });
const reply = (id, result) => process.stdout.write(JSON.stringify({ jsonrpc: "2.0", id, result }) + "\n");
rl.on("line", (line) => {
  let m; try { m = JSON.parse(line); } catch { return; }
  if (m.id === undefined) return; // notification
  switch (m.method) {
    case "initialize":
      return reply(m.id, { protocolVersion: m.params?.protocolVersion ?? "2025-06-18", capabilities: { tools: {} }, serverInfo: { name: "fake-mcp", version: "0.0.1" } });
    case "tools/list":
      return reply(m.id, { tools: [{ name: "echo", description: "Echo the input text back", inputSchema: { type: "object", properties: { text: { type: "string" } }, required: ["text"] } }] });
    case "tools/call":
      return reply(m.id, { content: [{ type: "text", text: String(m.params?.arguments?.text ?? "") }] });
    default:
      return process.stdout.write(JSON.stringify({ jsonrpc: "2.0", id: m.id, error: { code: -32601, message: "method not found" } }) + "\n");
  }
});
