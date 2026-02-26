package presets

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmd/search"
	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
)

// NewCmd creates the "presets" command.
func NewCmd() *cli.Command {
	return &cli.Command{
		Name:  "presets",
		Usage: "List available search preset filters",
		UsageText: `# List all presets in text format
  tp presets

  # List presets as JSON
  tp presets --output json`,
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmdutil.IsJSON(cmd) {
				type jsonPreset struct {
					Name        string `json:"name"`
					Description string `json:"description"`
					Where       string `json:"where"`
					Select      string `json:"select,omitempty"`
					OrderBy     string `json:"orderBy,omitempty"`
				}
				names := search.SortedPresetNames
				presetList := make([]jsonPreset, len(names))
				for i, name := range names {
					p := search.SearchPresets[name]
					presetList[i] = jsonPreset{
						Name:        p.Name,
						Description: p.Description,
						Where:       p.Where,
						Select:      p.Select,
						OrderBy:     p.OrderBy,
					}
				}
				return output.PrintJSON(os.Stdout, map[string]any{
					"presets": presetList,
				})
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(tw, "NAME\tDESCRIPTION\tWHERE\n")
			for _, name := range search.SortedPresetNames {
				p := search.SearchPresets[name]
				fmt.Fprintf(tw, "%s\t%s\t%s\n", name, p.Description, p.Where)
			}
			return tw.Flush()
		},
	}
}
