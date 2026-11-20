package plugin

import (
	"context"
	"encoding/json"
)

// ID is a unique runtime hot-path identifier of Plugin.
type ID uint64

// Name is a unique cold-path name of Plugin.
type Name string

// Plugin defines a runtime extension that initializes its resources within an isolated Slot.
type Plugin interface {
	// Init sets up the plugin using the provided configuration and registers its resources into the slot.
	Init(ctx context.Context, config InitConfig, slot *Slot) error
}

// Config holds information for plugin setup.
type Config = json.RawMessage

// InitConfig holds initialization parameters and metadata passed to a plugin during startup.
type InitConfig struct {
	Config Config
}
