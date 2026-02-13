package entity

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
	"github.com/lifedraft/targetprocess-cli/internal/text"
)

func newCreateCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new entity",
		UsageText: `# Create a new user story
  tp entity create --type UserStory --name "Implement login page" --project-id 42

  # Create a bug with description and team
  tp entity create --type Bug --name "Fix crash on startup" --project-id 42 --description "App crashes when..." --team-id 7

  # Create a task assigned to a user
  tp entity create --type Task --name "Write unit tests" --project-id 42 --assigned-user-id 15`,
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.StringFlag{Name: "type", Required: true, Usage: "Entity type (e.g. UserStory, Bug, Task, Feature)"},
			&cli.StringFlag{Name: "name", Required: true, Usage: "Entity name"},
			&cli.IntFlag{Name: "project-id", Required: true, Usage: "Project ID"},
			&cli.StringFlag{Name: "description", Usage: "Entity description"},
			&cli.IntFlag{Name: "team-id", Usage: "Team ID"},
			&cli.IntFlag{Name: "assigned-user-id", Usage: "Assigned user ID"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			entityType := cmd.String("type")
			projectID := cmd.Int("project-id")
			if projectID <= 0 {
				return fmt.Errorf("project ID must be positive, got %d", projectID)
			}

			fields := map[string]any{
				"Name":    cmd.String("name"),
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
