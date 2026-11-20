package builtin

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Skoowshot/qrix-runtime/internal/plugin"
)

type Echo struct{}

type echoConfig struct {
	InitMsg string `json:"init_msg"`
}

func (e *Echo) Init(ctx context.Context, config plugin.InitConfig, slot *plugin.Slot) error {
	var cfg echoConfig
	if len(config.Config) > 0 {
		if err := json.Unmarshal(config.Config, &cfg); err != nil {
			return fmt.Errorf("parse config: %w", err)
		}
	}

	log := slog.Default().With("plugin", "echo")
	msg := cmp.Or(cfg.InitMsg, "initialized")
	log.Info(msg)
	return nil
}
