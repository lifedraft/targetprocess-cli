package cmdutil

import (
	"sync"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/api"
	"github.com/lifedraft/targetprocess-cli/internal/config"
)

// Factory provides shared dependencies to all commands.
type Factory struct {
	ConfigPath string
	Debug      bool

	cfgOnce    sync.Once
	cfg        *config.Config
	cfgErr     error
	clientOnce sync.Once
	client     *api.Client
	clientErr  error
}

// Config returns the loaded configuration, caching after first load.
func (f *Factory) Config() (*config.Config, error) {
	f.cfgOnce.Do(func() {
		f.cfg, f.cfgErr = config.Load(f.ConfigPath)
	})
	return f.cfg, f.cfgErr
}

// Client returns an API client, creating one if needed.
func (f *Factory) Client() (*api.Client, error) {
	f.clientOnce.Do(func() {
		cfg, err := f.Config()
		if err != nil {
			f.clientErr = err
			return
		}
		if err := cfg.Validate(); err != nil {
			f.clientErr = err
			return
		}
		f.client = api.NewClient(cfg.Domain, cfg.Token, f.Debug)
	})
	return f.client, f.clientErr
}

// OutputFlag returns the standard --output flag for use in commands.
func OutputFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Value:   "text",
		Usage:   "Output format: text, json",
	}
}

// IsJSON returns true if the output format is JSON.
func IsJSON(cmd *cli.Command) bool {
	return cmd.String("output") == "json"
}
