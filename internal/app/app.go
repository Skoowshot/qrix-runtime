package app

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Skoowshot/qrix-runtime/internal/app/server"
	"github.com/Skoowshot/qrix-runtime/internal/app/service"
	"github.com/Skoowshot/qrix-runtime/internal/plugin"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
)

type App struct {
	servers []Server
}

func NewApp() *App {
	return &App{
		servers: make([]Server, 0),
	}
}

func (app *App) Start() error {
	wsRegistry := workspace.NewRegistry()
	plgRegistry := plugin.NewRegistry()

	wsService := service.NewWorkspace(15*time.Second, wsRegistry)
	plgService := service.NewPlugin(plgRegistry, wsRegistry)
	envService := service.NewEnvironment(wsRegistry)

	ctrlServer, err := server.ListenControl(":7178", wsService, plgService)
	if err != nil {
		return fmt.Errorf("create control server: %w", err)
	}

	envServer, err := server.ListenEnvironment(":7179", envService)
	if err != nil {
		return fmt.Errorf("create environment server: %w", err)
	}

	app.servers = append(
		app.servers,
		ctrlServer,
		envServer,
	)

	for _, s := range app.servers {
		go func() {
			if err := s.Start(); err != nil {
				slog.Error("failed to start server", "err", err)
			}
		}()
	}

	return nil
}

func (app *App) Stop(ctx context.Context) error {
	stopped := make(chan struct{})

	go func() {
		var wg sync.WaitGroup
		for _, s := range app.servers {
			wg.Add(1)

			go func(srv Server) {
				defer wg.Done()
				srv.GracefulStop()
			}(s)
		}
		wg.Wait()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		for _, s := range app.servers {
			s.Stop()
		}
		return ctx.Err()
	case <-stopped:
		return nil
	}
}
