package channels

import (
	"sync"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// ChannelFactory is a constructor function that creates a Channel from config and message bus.
// Each channel subpackage registers one or more factories via init().
type ChannelFactory func(cfg *config.Config, bus *bus.MessageBus) (Channel, error)

var (
	factoriesMu sync.RWMutex
	factories   = map[string]ChannelFactory{}

	// add custom factory map
	customMaps = map[string]string{}
)

// RegisterFactory registers a named channel factory. Called from subpackage init() functions.
func RegisterFactory(name string, f ChannelFactory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[name] = f
}

// getFactory looks up a channel factory by name.
func getFactory(name string) (ChannelFactory, bool) {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	f, ok := factories[name]
	return f, ok
}

// RegisterCustomFactory ...
func RegisterCustomFactory(name, displayName string, f ChannelFactory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[name] = f
	customMaps[name] = displayName
}

// UnregisterCustomFactories ...
func UnregisterCustomFactories(names ...string) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	for _, name := range names {
		delete(factories, name)
		delete(customMaps, name)
	}
}

// getCustoms looks up a channel factory by name.
func getCustoms() map[string]string {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	return customMaps
}
