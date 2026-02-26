package show

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
	"github.com/lifedraft/targetprocess-cli/internal/resolve"
)

// NewCmd creates the "show" command.
func NewCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:      "show",
		Usage:     "Show a Targetprocess entity by ID",
		ArgsUsage: "<id>",
		UsageText: `# Show an entity (auto-detects type)
  tp show 341079

  # Show with explicit type (skips auto-detection)
  tp show 341079 --type UserStory

  # Include related data
  tp show 341079 --include Project,Team

  # Output as JSON
  tp show 341079 -o json`,
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.StringFlag{Name: "type", Usage: "Entity type (auto-detected if omitted)"},
			&cli.StringFlag{Name: "include", Usage: "Related data to include, comma-separated (e.g. Project,Team)"},
			&cli.IntFlag{Name: "id", Usage: "Entity ID (alternative to positional argument)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := resolveID(cmd)
			if err != nil {
				return err
			}

			return RunShow(ctx, f, id, resolve.EntityType(cmd.String("type")), cmd.String("include"), cmdutil.IsJSON(cmd))
		},
	}
}

// RunShow executes the show logic. Exported so the root command can delegate to it.
func RunShow(ctx context.Context, f *cmdutil.Factory, id int, entityType, include string, jsonOutput bool) error {
	client, err := f.Client()
	if err != nil {
		return err
	}

	if entityType == "" {
		entityType, err = client.ResolveEntityType(ctx, id)
		if err != nil {
			return err
		}
	}

	var includes []string
	if include != "" {
		includes = strings.Split(include, ",")
	}

	entity, err := client.GetEntity(ctx, entityType, id, includes)
	if err != nil {
		return err
	}

	if jsonOutput {
		return output.PrintJSON(os.Stdout, entity)
	}

	output.PrintEntity(os.Stdout, entity)
	return nil
}

func resolveID(cmd *cli.Command) (int, error) {
	args := cmd.Args().Slice()
	if len(args) > 0 {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return 0, fmt.Errorf("invalid entity ID %q: must be an integer", args[0])
		}
		if id <= 0 {
			return 0, fmt.Errorf("entity ID must be positive, got %d", id)
		}
		return id, nil
	}

	if id := cmd.Int("id"); id > 0 {
		return id, nil
	}

	return 0, errors.New("entity ID is required; usage: tp show <id> or tp show --id <id>")
}
