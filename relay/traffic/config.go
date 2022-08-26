package traffic

type RelayConfig struct {
	MaxBodySize  int64  // Maximum length in bytes of relayed bodies.
	TargetHost   string // The host to relay traffic to. (e.g. 192.168.0.1:1234)
	TargetScheme string // The scheme ('http' or 'https') to use to communicate with the target host.
}

const DefaultMaxBodySize int64 = 1024 * 2048 // 2MB

func NewDefaultRelayConfig() *RelayConfig {
	return &RelayConfig{
		MaxBodySize: DefaultMaxBodySize,
	}
}
