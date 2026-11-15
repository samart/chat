package claude

import (
	"testing"

	"example.com/claudecli/wire"
)

func TestDecodeBridge(t *testing.T) {
	m := Message{Type: "result", Subtype: "success", Raw: []byte(`{"type":"result","subtype":"success","is_error":true,"result":"API Error","num_turns":1,"session_id":"s","uuid":"u","future":1}`)}
	v, err := m.Decode()
	if err != nil {
		t.Fatal(err)
	}
	r, ok := v.(*wire.ResultSuccess)
	if !ok || !r.IsError || r.Result != "API Error" {
		t.Fatalf("got %T %+v", v, v)
	}
	if _, err := (Message{Raw: []byte(`{"type":"brand_new"}`)}).Decode(); err == nil {
		t.Error("an unknown frame type should report an error, with Raw left intact")
	}
}
