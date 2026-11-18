package builtin

import (
	"context"

	"github.com/Skoowshot/qrix-runtime/internal/plugin"
)

type Echo struct{}

type echoConfig struct {
	Msg string `json:"msg"`
}

func (e *Echo) Init(ctx context.Context, config plugin.InitConfig, slot *plugin.Slot) error {
	return nil
}
