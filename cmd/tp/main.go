package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/urfave/cli/v3"

	apicmd "github.com/lifedraft/targetprocess-cli/internal/cmd/api"
	"github.com/lifedraft/targetprocess-cli/internal/cmd/bugreport"
	cheatsht "github.com/lifedraft/targetprocess-cli/internal/cmd/cheatsheet"
	"github.com/lifedraft/targetprocess-cli/internal/cmd/commentcmd"
	configcmd "github.com/lifedraft/targetprocess-cli/internal/cmd/config"
	createcmd "github.com/lifedraft/targetprocess-cli/internal/cmd/create"
	"github.com/lifedraft/targetprocess-cli/internal/cmd/inspect"
	"github.com/lifedraft/targetprocess-cli/internal/cmd/presets"
	querycmd "github.com/lifedraft/targetprocess-cli/internal/cmd/query"
	searchcmd "github.com/lifedraft/targetprocess-cli/internal/cmd/search"
	showcmd "github.com/lifedraft/targetprocess-cli/internal/cmd/show"
	updatecmd "github.com/lifedraft/targetprocess-cli/internal/cmd/update"
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

	showCmd := showcmd.NewCmd(f)
	searchCmd := searchcmd.NewCmd(f)
	createCmd := createcmd.NewCmd(f)
	updateCmd := updatecmd.NewCmd(f)
	commentCmd := commentcmd.NewCmd(f)

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
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) == 0 {
				return cli.ShowAppHelp(cmd)
			}

			// If the first arg is a positive integer, delegate to "show"
			id, err := strconv.Atoi(args[0])
			if err == nil && id > 0 {
				return showcmd.RunShow(ctx, f, id, "", "", false)
			}

			return cli.ShowAppHelp(cmd)
		},
		Commands: []*cli.Command{
			showCmd,
			searchCmd,
			createCmd,
			updateCmd,
			commentCmd,
			presets.NewCmd(),
			querycmd.NewCmd(f),
			inspect.NewCmd(f),
			apicmd.NewCmd(f),
			configcmd.NewCmd(f),
			cheatsht.NewCmd(f),
			bugreport.NewCmd(f, version),

			// Hidden aliases
			hiddenAlias("get", "show", showCmd),
			hiddenAlias("view", "show", showCmd),
			hiddenAlias("find", "search", searchCmd),
			hiddenAlias("list", "search", searchCmd),
			hiddenAlias("edit", "update", updateCmd),
			hiddenAlias("new", "create", createCmd),
			hiddenAlias("add", "create", createCmd),
			hiddenAlias("comments", "comment", commentCmd),
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

// hiddenAlias creates a hidden command that delegates to the target command.
func hiddenAlias(alias, target string, targetCmd *cli.Command) *cli.Command {
	return &cli.Command{
		Name:      alias,
		Hidden:    true,
		Usage:     targetCmd.Usage,
		ArgsUsage: targetCmd.ArgsUsage,
		Flags:     targetCmd.Flags,
		Commands:  targetCmd.Commands,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Fprintf(os.Stderr, "Hint: %q is an alias for %q\n", alias, target)
			return targetCmd.Action(ctx, cmd)
		},
	}
}
