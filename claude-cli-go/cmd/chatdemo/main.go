// Command chatdemo is a minimal terminal chat built on package claude. It
// shows the pieces a real host needs: one long-lived CLI process, streaming
// text, tool approvals, interrupt and model switching.
//
//	go run ./cmd/chatdemo -bin /path/to/claude [-cwd dir] [-resume id] [-model opus]
//
// Type a message and press Enter. Commands: /stop (interrupt), /model <name>,
// /mode <mode>, /quit. Anything else starting with "/" is sent to the CLI as
// a slash command.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"example.com/claudecli/claude"
)

func main() {
	bin := flag.String("bin", "claude", "path to the claude CLI")
	cwd := flag.String("cwd", ".", "working directory for the session")
	resume := flag.String("resume", "", "session id to resume")
	model := flag.String("model", "", "model alias or id")
	flag.Parse()

	lines := make(chan string)
	go func() {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()

	opts := claude.Options{
		PathToClaudeCodeExecutable: *bin,
		Cwd:                        *cwd,
		Resume:                     *resume,
		Model:                      *model,
		IncludePartialMessages:     true,
		SystemPrompt:               claude.PresetSystemPrompt(""),
		SettingSources:             []string{"user", "project", "local"},
		InitializeTimeout:          60 * time.Second,
		Stderr:                     func(s string) { fmt.Fprint(os.Stderr, dim(s)) },
		CanUseTool: func(ctx context.Context, req claude.PermissionRequest) (*claude.PermissionResult, error) {
			fmt.Printf("\n%s wants to run %s\n  %s\nAllow? [y/N/feedback] ", who(req), req.ToolName, oneLine(req.Input))
			select {
			case answer, ok := <-lines:
				switch a := strings.TrimSpace(answer); {
				case !ok:
					return claude.Deny("input closed"), nil
				case strings.EqualFold(a, "y"), strings.EqualFold(a, "yes"):
					return claude.Allow(req.Input), nil
				case a == "", strings.EqualFold(a, "n"):
					return claude.Deny("The user declined this action."), nil
				default:
					return claude.Deny(a), nil // feedback goes back to Claude
				}
			case <-ctx.Done(): // the CLI withdrew the request, e.g. after an interrupt
				fmt.Println("(request cancelled)")
				return claude.Deny("cancelled"), nil
			}
		},
	}

	ctx := context.Background()
	q, err := claude.Start(ctx, opts)
	if err != nil {
		fatal(err)
	}
	defer q.Close()

	init, err := q.Initialized(ctx)
	if err != nil {
		fatal(err)
	}
	var info struct {
		Commands []struct{ Name string }  `json:"commands"`
		Models   []struct{ Value string } `json:"models"`
	}
	_ = json.Unmarshal(init, &info)
	fmt.Printf("connected (pid %d): %d slash commands, %d models\n> ", q.PID(), len(info.Commands), len(info.Models))

	busy := false
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				return
			}
			line = strings.TrimSpace(line)
			switch {
			case line == "":
			case line == "/quit":
				return
			case line == "/stop":
				if _, err := q.Interrupt(ctx, false); err != nil {
					fmt.Println("interrupt:", err)
				}
			case strings.HasPrefix(line, "/model "):
				report(q.SetModel(ctx, strings.TrimSpace(line[7:])))
			case strings.HasPrefix(line, "/mode "):
				report(q.SetPermissionMode(ctx, strings.TrimSpace(line[6:])))
			default:
				if err := q.SendText(line); err != nil {
					fatal(err)
				}
				busy = true
			}
			if !busy {
				fmt.Print("> ")
			}

		case m, ok := <-q.Messages():
			if !ok {
				if err := q.Err(); err != nil {
					fatal(err)
				}
				return
			}
			switch m.Type {
			case "stream_event":
				var ev struct {
					Event struct {
						Type  string `json:"type"`
						Delta struct {
							Type string `json:"type"`
							Text string `json:"text"`
						} `json:"delta"`
					} `json:"event"`
				}
				if json.Unmarshal(m.Raw, &ev) == nil && ev.Event.Delta.Type == "text_delta" && m.ParentToolUseID == "" {
					fmt.Print(ev.Event.Delta.Text)
				}
			case "assistant":
				var a struct {
					Message struct {
						Content []struct {
							Type  string          `json:"type"`
							Name  string          `json:"name"`
							Input json.RawMessage `json:"input"`
						} `json:"content"`
					} `json:"message"`
				}
				_ = json.Unmarshal(m.Raw, &a)
				for _, b := range a.Message.Content {
					if b.Type == "tool_use" {
						fmt.Printf("\n%s\n", dim("["+b.Name+"] "+oneLine(b.Input)))
					}
				}
			case "result":
				var r struct {
					IsError bool    `json:"is_error"`
					Result  string  `json:"result"`
					Cost    float64 `json:"total_cost_usd"`
					Turns   int     `json:"num_turns"`
				}
				_ = json.Unmarshal(m.Raw, &r)
				if r.IsError { // check is_error, not subtype: API errors arrive as subtype "success"
					fmt.Printf("\nerror (%s): %s\n", m.Subtype, r.Result)
				} else {
					fmt.Printf("\n%s\n", dim(fmt.Sprintf("(%d turns, $%.4f)", r.Turns, r.Cost)))
				}
				busy = false
				fmt.Print("> ")
			case "system", "user":
				// init, status, task_* and tool results: a real UI renders these.
			default:
				fmt.Printf("\n%s\n", dim("(unhandled frame type "+m.Type+")")) // tolerant reader
			}
		}
	}
}

func who(req claude.PermissionRequest) string {
	if req.AgentID != "" {
		return "Subagent " + req.AgentID
	}
	return "Claude"
}

func oneLine(raw json.RawMessage) string {
	s := strings.Join(strings.Fields(string(raw)), " ")
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

func dim(s string) string { return "\x1b[2m" + s + "\x1b[0m" }

func report(err error) {
	if err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Println("ok")
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "chatdemo:", err)
	os.Exit(1)
}
