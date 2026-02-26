package create

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
	"github.com/lifedraft/targetprocess-cli/internal/resolve"
	"github.com/lifedraft/targetprocess-cli/internal/text"
)

// NewCmd creates the "create" command.
func NewCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a new entity",
		ArgsUsage: "<type> <name>",
		UsageText: `# Create a new user story
  tp create UserStory "Implement login page" --project-id 42

  # Create a bug with description and team
  tp create Bug "Fix crash on startup" --project-id 42 --description "App crashes when..."

  # Create a task assigned to a user
  tp create Task "Write unit tests" --project-id 42 --assigned-user-id 15`,
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.IntFlag{Name: "project-id", Required: true, Usage: "Project ID"},
			&cli.StringFlag{Name: "description", Usage: "Entity description"},
			&cli.IntFlag{Name: "team-id", Usage: "Team ID"},
			&cli.IntFlag{Name: "assigned-user-id", Usage: "Assigned user ID"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) < 2 {
				return errors.New("entity type and name are required; usage: tp create <type> <name>")
			}

			entityType := resolve.EntityType(args[0])
			name := args[1]

			client, err := f.Client()
			if err != nil {
				return err
			}

			projectID := cmd.Int("project-id")
			if projectID <= 0 {
				return fmt.Errorf("project ID must be positive, got %d", projectID)
			}

			fields := map[string]any{
				"Name":    name,
				"Project": map[string]any{"Id": projectID},
			}

			if desc := cmd.String("description"); desc != "" {
				fields["Description"] = desc
			}
			if teamID := cmd.Int("team-id"); teamID > 0 {
				fields["Team"] = map[string]any{"Id": teamID}
			}
			if userID := cmd.Int("assigned-user-id"); userID > 0 {
				fields["AssignedUser"] = map[string]any{"Id": userID}
			}

			if prepErr := text.PrepareFields(ctx, client, fields); prepErr != nil {
				return prepErr
			}

			entity, err := client.CreateEntity(ctx, entityType, fields)
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
