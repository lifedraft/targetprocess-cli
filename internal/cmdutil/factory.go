package cmdutil

import (
	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/api"
	"github.com/lifedraft/targetprocess-cli/internal/config"
)

// Factory provides shared dependencies to all commands.
type Factory struct {
	ConfigPath string
	Debug      bool

	cfg    *config.Config
	client *api.Client
}

// Config returns the loaded configuration, caching after first load.
func (f *Factory) Config() (*config.Config, error) {
	if f.cfg != nil {
		return f.cfg, nil
	}
	cfg, err := config.Load(f.ConfigPath)
	if err != nil {
		return nil, err
	}
	f.cfg = cfg
	return cfg, nil
}

// Client returns an API client, creating one if needed.
func (f *Factory) Client() (*api.Client, error) {
	if f.client != nil {
		return f.client, nil
	}
	cfg, err := f.Config()
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	f.client = api.NewClient(cfg.Domain, cfg.Token, f.Debug)
	return f.client, nil
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
