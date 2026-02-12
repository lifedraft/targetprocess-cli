package entity

import (
	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
)

func NewCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "entity",
		Usage: "Manage Targetprocess entities (search, get, create, update)",
		UsageText: `# Search for open user stories
  tp entity search --type UserStory --preset open

  # Get a specific entity by ID
  tp entity get --type UserStory --id 12345

  # Create a new bug
  tp entity create --type Bug --name "Login fails on Safari" --project-id 42

  # Update an entity's state
  tp entity update --type UserStory --id 12345 --state-id 100

  # List available search presets
  tp entity presets`,
		Commands: []*cli.Command{
			newSearchCmd(f),
			newGetCmd(f),
			newCreateCmd(f),
			newUpdateCmd(f),
			newPresetsCmd(),
		},
	}
}
