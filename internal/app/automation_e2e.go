//go:build e2e

package app

// The development driver starts only when MYGO_AUTOMATION explicitly names a socket.
import _ "github.com/egoist/mygo/plugins/automation"
