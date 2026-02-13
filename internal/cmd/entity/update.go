package entity

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
	"github.com/lifedraft/targetprocess-cli/internal/text"
)

func newUpdateCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update an existing entity",
		UsageText: `# Rename an entity
  tp entity update --type UserStory --id 12345 --name "Updated story title"

  # Change entity state
  tp entity update --type Bug --id 67890 --state-id 100

  # Reassign and update description
  tp entity update --type Task --id 111 --assigned-user-id 15 --description "Updated requirements"`,
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.StringFlag{Name: "type", Required: true, Usage: "Entity type (e.g. UserStory, Bug, Task, Feature)"},
			&cli.IntFlag{Name: "id", Required: true, Usage: "Entity ID"},
			&cli.StringFlag{Name: "name", Usage: "New name"},
			&cli.StringFlag{Name: "description", Usage: "New description"},
			&cli.IntFlag{Name: "state-id", Usage: "New entity state ID"},
			&cli.IntFlag{Name: "assigned-user-id", Usage: "New assigned user ID"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			entityType := cmd.String("type")
			id := cmd.Int("id")
			if id <= 0 {
				return fmt.Errorf("entity ID must be positive, got %d", id)
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

			if err := text.PrepareFields(ctx, client, fields); err != nil {
				return err
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
