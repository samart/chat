package claude

import (
	"context"
	"encoding/json"
	"os"
	"time"
)

// manifestCaptureDeadline is the SDK's overall budget for the capture below.
const manifestCaptureDeadline = 250 * time.Millisecond

// captureMcpManifests runs a local MCP handshake with each in-process server
// before initialize is sent, and returns
//
//	{name: {"initializeResult": …, "toolsListResult": …}}
//
// for the initialize request's sdkMcpServerManifests. The SDK does this so the
// CLI knows each server's capabilities and tools without a round trip. Servers
// that do not answer inside the deadline are left out; the CLI then performs
// the handshake itself over mcp_message.
//
// It is skipped when CLAUDE_AGENT_SDK_DISABLE_MCP_MANIFESTS is truthy.
func captureMcpManifests(ctx context.Context, servers []SdkMcpServer, sdkVersion string) orderedObject {
	if envTruthy(os.Getenv("CLAUDE_AGENT_SDK_DISABLE_MCP_MANIFESTS")) {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, manifestCaptureDeadline)
	defer cancel()

	initReq, _ := marshalJSON(orderedObject{
		{"jsonrpc", "2.0"},
		{"id", "sdk-manifest-capture:initialize"},
		{"method", "initialize"},
		{"params", orderedObject{
			{"protocolVersion", "2025-11-25"},
			{"capabilities", struct{}{}},
			{"clientInfo", orderedObject{
				{"name", "claude-code"},
				{"title", "Claude Code"},
				{"description", "Anthropic's agentic coding tool"},
				{"websiteUrl", "https://claude.com/claude-code"},
				{"version", sdkVersion},
			}},
		}},
	})
	initialized := json.RawMessage(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	toolsList := json.RawMessage(`{"jsonrpc":"2.0","id":"sdk-manifest-capture:tools-list","method":"tools/list"}`)

	type result struct {
		name     string
		manifest orderedObject
	}
	results := make(chan result, len(servers))
	for _, s := range servers {
		go func(s SdkMcpServer) {
			res := result{name: s.Name}
			defer func() {
				_ = recover() // a panicking server simply has no manifest
				results <- res
			}()
			initRes := rpcResult(s.Handle(ctx, initReq))
			if initRes == nil || ctx.Err() != nil {
				return
			}
			_, _ = s.Handle(ctx, initialized)
			manifest := orderedObject{{"initializeResult", initRes}}
			var caps struct {
				Capabilities struct {
					Tools json.RawMessage `json:"tools"`
				} `json:"capabilities"`
			}
			if json.Unmarshal(initRes, &caps) == nil && len(caps.Capabilities.Tools) > 0 && string(caps.Capabilities.Tools) != "null" {
				if tools := rpcResult(s.Handle(ctx, toolsList)); tools != nil && ctx.Err() == nil {
					var page struct {
						NextCursor *string `json:"nextCursor"`
					}
					// Only a complete, single-page listing is useful.
					if json.Unmarshal(tools, &page) == nil && page.NextCursor == nil {
						manifest = append(manifest, InitField{"toolsListResult", tools})
					}
				}
			}
			res.manifest = manifest
		}(s)
	}

	byName := map[string]orderedObject{}
	for range servers {
		select {
		case r := <-results:
			if r.manifest != nil {
				byName[r.name] = r.manifest
			}
		case <-ctx.Done():
		}
	}
	var out orderedObject
	for _, s := range servers { // keep the caller's server order
		if m, ok := byName[s.Name]; ok {
			out = append(out, InitField{s.Name, m})
		}
	}
	return out
}

// rpcResult extracts "result" from a JSON-RPC response, or nil.
func rpcResult(resp json.RawMessage, err error) json.RawMessage {
	if err != nil || len(resp) == 0 {
		return nil
	}
	var r struct {
		Result json.RawMessage `json:"result"`
	}
	if json.Unmarshal(resp, &r) != nil || len(r.Result) == 0 {
		return nil
	}
	return r.Result
}
