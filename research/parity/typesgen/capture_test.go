package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCaptures(t *testing.T) {
	files, _ := filepath.Glob(os.Getenv("CAPTURES") + "/*/*/transcript.ndjson")
	more, _ := filepath.Glob(os.Getenv("CAPTURES") + "/*/transcript.ndjson")
	files = append(files, more...)
	kinds := map[string]int{}
	unknown := map[string]int{}
	n := 0
	for _, f := range files {
		fh, _ := os.Open(f)
		sc := bufio.NewScanner(fh)
		sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
		for sc.Scan() {
			var env struct {
				Msg json.RawMessage `json:"msg"`
			}
			if json.Unmarshal(sc.Bytes(), &env) != nil || len(env.Msg) == 0 || env.Msg[0] != '{' {
				continue
			}
			n++
			var m StdoutMessage
			if err := json.Unmarshal(env.Msg, &m); err != nil {
				t.Fatalf("%s: %v", f, err)
			}
			out, _ := json.Marshal(m)
			semEq(t, env.Msg, out)
			key := m.Type + ":" + m.Subtype
			v, err := m.Decode()
			if err != nil {
				unknown[key]++
				continue
			}
			kinds[key]++
			out2, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			semEq(t, env.Msg, out2)
			if cr, ok := v.(*ControlRequest); ok {
				iv, err := cr.Request.Decode()
				if err != nil {
					unknown["request:"+cr.Request.Subtype]++
					continue
				}
				kinds["request:"+cr.Request.Subtype]++
				b, _ := json.Marshal(iv)
				var orig struct{ Request json.RawMessage `json:"request"` }
				_ = json.Unmarshal(env.Msg, &orig)
				semEq(t, orig.Request, b)
			}
		}
		fh.Close()
	}
	t.Logf("frames=%d files=%d", n, len(files))
	t.Logf("decoded: %v", kinds)
	t.Logf("unknown (kept raw): %v", unknown)
}
