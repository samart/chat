package claude

import (
	"encoding/json"

	"example.com/claudecli/wire"
)

// Wire returns the frame as a wire.Message union, which keeps Raw intact and
// offers typed As<Variant>() accessors.
func (m Message) Wire() (wire.Message, error) {
	var w wire.Message
	err := json.Unmarshal(m.Raw, &w)
	return w, err
}

// Decode returns the frame as its typed Go value, for example
// *wire.AssistantMessage, *wire.ResultSuccess or *wire.TaskStartedMessage.
// A frame type this package does not know yields an error; the frame's Raw
// JSON is still complete, so forward or log it rather than dropping it.
func (m Message) Decode() (any, error) {
	w, err := m.Wire()
	if err != nil {
		return nil, err
	}
	return w.Decode()
}
