package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/urfave/cli/v3"

	apicmd "github.com/lifedraft/targetprocess-cli/internal/cmd/api"
	"github.com/lifedraft/targetprocess-cli/internal/cmd/bugreport"
	cheatsht "github.com/lifedraft/targetprocess-cli/internal/cmd/cheatsheet"
	configcmd "github.com/lifedraft/targetprocess-cli/internal/cmd/config"
	"github.com/lifedraft/targetprocess-cli/internal/cmd/entity"
	"github.com/lifedraft/targetprocess-cli/internal/cmd/inspect"
	querycmd "github.com/lifedraft/targetprocess-cli/internal/cmd/query"
	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
)

var version = "dev"

func main() {
	os.Exit(run())
}

func run() (exitCode int) {
	f := &cmdutil.Factory{}

	defer func() {
		if r := recover(); r != nil {
			bugreport.HandlePanic(f, version, r)
			exitCode = 2
		}
	}()

	root := &cli.Command{
		Name:    "tp",
		Usage:   "Targetprocess CLI - interact with Targetprocess from the command line",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "Path to config file",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Enable debug output to stderr",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			f.ConfigPath = cmd.String("config")
			f.Debug = cmd.Bool("debug")
			return ctx, nil
		},
		Commands: []*cli.Command{
			querycmd.NewCmd(f),
			entity.NewCmd(f),
			inspect.NewCmd(f),
			apicmd.NewCmd(f),
			configcmd.NewCmd(f),
			cheatsht.NewCmd(f),
			bugreport.NewCmd(f, version),
		},
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

	err := root.Run(ctx, os.Args)
	cancel()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return 130
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}
