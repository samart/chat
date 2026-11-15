package wire

import (
	"os"
	"path/filepath"
)

// capturesRoot is the directory of recorded CLI runs used by the round-trip
// tests: $CAPTURES, or the 2.1.311 recordings in this repository.
func capturesRoot() string {
	if d := os.Getenv("CAPTURES"); d != "" {
		return d
	}
	return filepath.Join("..", "..", "verification", "runs", "v2.1.311")
}
