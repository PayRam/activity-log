package httpmiddleware

import "sync"

var (
	defaultConfigMu sync.RWMutex
	defaultConfig   *Config
)

// SetDefaultConfig sets the package-level default config used by Middleware() with no args.
func SetDefaultConfig(cfg Config) {
	cfgCopy := cfg
	defaultConfigMu.Lock()
	defaultConfig = &cfgCopy
	defaultConfigMu.Unlock()
}

// ResetDefaultConfig clears the package-level default config.
func ResetDefaultConfig() {
	defaultConfigMu.Lock()
	defaultConfig = nil
	defaultConfigMu.Unlock()
}

func loadDefaultConfig() (Config, bool) {
	defaultConfigMu.RLock()
	defer defaultConfigMu.RUnlock()
	if defaultConfig == nil {
		return Config{}, false
	}
	return *defaultConfig, true
}
