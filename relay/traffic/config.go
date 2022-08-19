package traffic

import "regexp"

type RelayConfig struct {
	MaxBodySize    int64           // Maximum length in bytes of relayed bodies.
	OriginOverride string          // Default to passing Origin header as-is, but set to override.
	RelayedCookies map[string]bool // The name of cookies that should be relayed.
	SpecialPaths   []SpecialPath   // Path patterns that are mapped to fully qualified URLs.
	TargetHost     string          // The host to relay traffic to. (e.g. 192.168.0.1:1234)
	TargetScheme   string          // The scheme ('http' or 'https') to use to communicate with the target host.
}

type SpecialPath struct {
	Match       *regexp.Regexp
	Replacement string
}

const DefaultMaxBodySize int64 = 1024 * 2048 // 2MB

func NewDefaultRelayConfig() *RelayConfig {
	return &RelayConfig{
		MaxBodySize:    DefaultMaxBodySize,
		RelayedCookies: map[string]bool{},
	}
}
