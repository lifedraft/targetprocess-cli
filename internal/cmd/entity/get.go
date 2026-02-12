package entity

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
)

func newGetCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a specific entity by ID",
		UsageText: `# Get a user story by ID
  tp entity get --type UserStory --id 12345

  # Get a bug with related project and team data
  tp entity get --type Bug --id 67890 --include Project,Team

  # Get entity details as JSON
  tp entity get --type Feature --id 111 --output json`,
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.StringFlag{Name: "type", Required: true, Usage: "Entity type (e.g. UserStory, Bug, Task, Feature)"},
			&cli.IntFlag{Name: "id", Required: true, Usage: "Entity ID"},
			&cli.StringFlag{Name: "include", Usage: "Related data to include, comma-separated (e.g. Project,Team)"},
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

			var includes []string
			if inc := cmd.String("include"); inc != "" {
				includes = strings.Split(inc, ",")
			}

			entity, err := client.GetEntity(ctx, entityType, id, includes)
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
