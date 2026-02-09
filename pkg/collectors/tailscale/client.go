package tailscale

import (
	"tailscale.com/client/local"
)

// newRealClient creates a *local.Client configured with the given socket path.
// If socketPath is empty, the platform default is used.
func newRealClient(socketPath string) *local.Client {
	lc := &local.Client{}
	if socketPath != "" {
		lc.Socket = socketPath
	}
	return lc
}
