package traffic

// RelayOptions contains configuration options for the core relay code.
//
// It's preferable to keep the core relay code simple; before adding a new
// option here, consider whether you could implement the same functionality as a
// plugin.
type RelayOptions struct {
	MaxBodySize  int64  // Maximum length in bytes of relayed bodies.
	TargetHost   string // The host to relay traffic to. (e.g. 192.168.0.1:1234)
	TargetScheme string // The scheme ('http' or 'https') to use to communicate with the target host.
}

const DefaultMaxBodySize int64 = 1024 * 2048 // 2MB

func NewDefaultRelayOptions() *RelayOptions {
	return &RelayOptions{
		MaxBodySize: DefaultMaxBodySize,
	}
}
