package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
	"github.com/lifedraft/targetprocess-cli/internal/resolve"
	"github.com/lifedraft/targetprocess-cli/internal/text"
)

// NewCmd creates the "update" command.
func NewCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "Update an existing entity",
		ArgsUsage: "<id>",
		UsageText: `# Rename an entity (auto-detects type)
  tp update 12345 --name "Updated story title"

  # Change entity state
  tp update 67890 --state-id 100

  # Update with explicit type (skips auto-detection)
  tp update 111 --type Task --assigned-user-id 15 --description "Updated requirements"`,
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.StringFlag{Name: "type", Usage: "Entity type (auto-detected if omitted)"},
			&cli.IntFlag{Name: "id", Usage: "Entity ID (alternative to positional argument)"},
			&cli.StringFlag{Name: "name", Usage: "New name"},
			&cli.StringFlag{Name: "description", Usage: "New description"},
			&cli.IntFlag{Name: "state-id", Usage: "New entity state ID"},
			&cli.IntFlag{Name: "assigned-user-id", Usage: "New assigned user ID"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := resolveID(cmd)
			if err != nil {
				return err
			}

			client, err := f.Client()
			if err != nil {
				return err
			}

			entityType := resolve.EntityType(cmd.String("type"))
			if entityType == "" {
				entityType, err = client.ResolveEntityType(ctx, id)
				if err != nil {
					return err
				}
			}

			fields := map[string]any{}

			if name := cmd.String("name"); name != "" {
				fields["Name"] = name
			}
			if desc := cmd.String("description"); desc != "" {
				fields["Description"] = desc
			}
			if stateID := cmd.Int("state-id"); stateID > 0 {
				fields["EntityState"] = map[string]any{"Id": stateID}
			}
			if userID := cmd.Int("assigned-user-id"); userID > 0 {
				fields["AssignedUser"] = map[string]any{"Id": userID}
			}

			if len(fields) == 0 {
				return errors.New("no fields to update; specify at least one of --name, --description, --state-id, or --assigned-user-id")
			}

			if prepErr := text.PrepareFields(ctx, client, fields); prepErr != nil {
				return prepErr
			}

			entity, err := client.UpdateEntity(ctx, entityType, id, fields)
			if err != nil {
				return err
			}

			if cmdutil.IsJSON(cmd) {
				return output.PrintJSON(os.Stdout, entity)
			}

			output.PrintEntity(os.Stdout, entity)
			return nil
		},
	}
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

	return 0, errors.New("entity ID is required; usage: tp update <id> or tp update --id <id>")
}
