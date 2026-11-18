package builtin

import (
	"context"

	"github.com/Skoowshot/qrix-runtime/internal/plugin"
)

type Echo struct{}

type echoConfig struct {
	InitMsg string `json:"init_msg"`
}

func (e *Echo) Init(ctx context.Context, config plugin.InitConfig, slot *plugin.Slot) error {
	return nil
}
