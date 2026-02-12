package config

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	internalconfig "github.com/lifedraft/targetprocess-cli/internal/config"
	"github.com/lifedraft/targetprocess-cli/internal/output"
)

func NewCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Manage CLI configuration",
		Commands: []*cli.Command{
			newGetCmd(f),
			newSetCmd(f),
			newListCmd(f),
			newPathCmd(),
		},
	}
}

func newGetCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "Get a config value",
		ArgsUsage: "<key>",
		Flags:     []cli.Flag{cmdutil.OutputFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			key := cmd.Args().First()
			if key == "" {
				return errors.New("key argument is required (valid keys: domain, token)")
			}
			if key == "token" {
				cfg, err := internalconfig.Load(f.ConfigPath)
				if err != nil {
					return err
				}
				configured := cfg.Token != ""
				if cmdutil.IsJSON(cmd) {
					return output.PrintJSON(os.Stdout, map[string]any{
						"configured": configured,
						"source":     string(cfg.TokenSource),
					})
				}
				if configured {
					fmt.Printf("Token is configured (source: %s)\n", cfg.TokenSource)
				} else {
					fmt.Println("Token is not configured")
				}
				return nil
			}
			val, err := internalconfig.Get(f.ConfigPath, key)
			if err != nil {
				return err
			}
			if cmdutil.IsJSON(cmd) {
				return output.PrintJSON(os.Stdout, map[string]string{key: val})
			}
			fmt.Println(val)
			return nil
		},
	}
}

func newSetCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:      "set",
		Usage:     "Set a config value",
		ArgsUsage: "<key> <value>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 2 {
				return errors.New("usage: tp config set <key> <value>")
			}
			key := cmd.Args().Get(0)
			value := cmd.Args().Get(1)

			if key == "token" {
				source, err := internalconfig.SetToken(f.ConfigPath, value)
				if err != nil {
					return err
				}
				switch source {
				case internalconfig.TokenSourceKeyring:
					fmt.Fprintln(os.Stderr, "Token stored in system keychain")
				case internalconfig.TokenSourceFile:
					fmt.Fprintf(os.Stderr, "Warning: keychain unavailable, token stored in plain text at %s\n", internalconfig.DefaultPath())
				case internalconfig.TokenSourceNone, internalconfig.TokenSourceEnv:
					// Not reachable from SetToken, but satisfy exhaustive check.
				}
				return nil
			}

			if err := internalconfig.Set(f.ConfigPath, key, value); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Set %s successfully\n", key)
			return nil
		},
	}
}

func newListCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all config values",
		Flags: []cli.Flag{cmdutil.OutputFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := internalconfig.Load(f.ConfigPath)
			if err != nil {
				return err
			}
			token := redactToken(cfg.Token)
			source := string(cfg.TokenSource)
			if cmdutil.IsJSON(cmd) {
				return output.PrintJSON(os.Stdout, map[string]string{
					"domain":       cfg.Domain,
					"token":        token,
					"token_source": source,
				})
			}
			fmt.Printf("domain: %s\n", cfg.Domain)
			fmt.Printf("token:  %s (source: %s)\n", token, source)
			return nil
		},
	}
}

// redactToken masks a token for display, showing only the first 4 and last 4 characters.
func redactToken(token string) string {
	if len(token) > 8 {
		return token[:4] + "..." + token[len(token)-4:]
	}
	if token != "" {
		return "[REDACTED]"
	}
	return ""
}

func newPathCmd() *cli.Command {
	return &cli.Command{
		Name:  "path",
		Usage: "Show the config file path",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println(internalconfig.DefaultPath())
			return nil
		},
	}
}
