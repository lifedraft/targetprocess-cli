package entity

import (
	"context"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
)

// preset defines a reusable search filter with optional field projection and sorting.
type preset struct {
	Name        string
	Description string
	Where       string
	Select      string // optional v2 select expression
	OrderBy     string // optional v2 orderBy expression
}

var searchPresets = map[string]preset{
	// Status-based
	"open": {
		Name:        "open",
		Description: "Entities in initial (open) state",
		Where:       "entityState.isInitial==true",
	},
	"inProgress": {
		Name:        "inProgress",
		Description: "Entities in a planned (in-progress) state",
		Where:       "entityState.isPlanned==true",
	},
	"done": {
		Name:        "done",
		Description: "Entities in final (done) state",
		Where:       "entityState.isFinal==true",
	},

	// Assignment-based
	"unassigned": {
		Name:        "unassigned",
		Description: "Entities with no assignments",
		Where:       "assignments.count==0",
	},

	// Priority-based
	"highPriority": {
		Name:        "highPriority",
		Description: "High-priority entities (importance >= 90)",
		Where:       "priority.importance>=90",
	},

	// Time-based
	"createdToday": {
		Name:        "createdToday",
		Description: "Entities created today",
		Where:       "createDate>=Today",
	},
	"modifiedToday": {
		Name:        "modifiedToday",
		Description: "Entities modified today",
		Where:       "modifyDate>=Today",
	},
	"createdThisWeek": {
		Name:        "createdThisWeek",
		Description: "Entities created in the last 7 days",
		Where:       "createDate>=Today.AddDays(-7)",
	},
	"modifiedThisWeek": {
		Name:        "modifiedThisWeek",
		Description: "Entities modified in the last 7 days",
		Where:       "modifyDate>=Today.AddDays(-7)",
	},
	"createdLastWeek": {
		Name:        "createdLastWeek",
		Description: "Entities created between 14 and 7 days ago",
		Where:       "createDate>=Today.AddDays(-14) and createDate<Today.AddDays(-7)",
	},
	"modifiedLastWeek": {
		Name:        "modifiedLastWeek",
		Description: "Entities modified between 14 and 7 days ago",
		Where:       "modifyDate>=Today.AddDays(-14) and modifyDate<Today.AddDays(-7)",
	},

	// Combined
	"highPriorityUnassigned": {
		Name:        "highPriorityUnassigned",
		Description: "High-priority entities with no assignments",
		Where:       "priority.importance>=90 and assignments.count==0",
	},

	// v2-powered presets with select and orderBy
	"recentActivity": {
		Name:        "recentActivity",
		Description: "Recently modified entities (last 7 days), sorted by modification date",
		Where:       "modifyDate>=Today.AddDays(-7)",
		Select:      "id,name,entityType.name as type,entityState.name as state,modifyDate",
		OrderBy:     "modifyDate desc",
	},
	"sprintStatus": {
		Name:        "sprintStatus",
		Description: "Active sprint items that are not yet done",
		Where:       "teamIteration!=null and entityState.isFinal!=true",
		Select:      "id,name,entityType.name as type,entityState.name as state,effort",
		OrderBy:     "entityState.name",
	},
	"unestimated": {
		Name:        "unestimated",
		Description: "Unestimated entities that are not yet done",
		Where:       "(effort==null or effort==0) and entityState.isFinal!=true",
		Select:      "id,name,entityState.name as state",
	},
}

// sortedPresetNames is the sorted list of preset names, computed once.
var sortedPresetNames = func() []string {
	names := make([]string, 0, len(searchPresets))
	for name := range searchPresets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}()

// applyPreset resolves a preset name into a full preset struct.
// If where is also provided, the preset where and the extra where are combined with " and ".
func applyPreset(presetName, where string) (preset, error) {
	p, ok := searchPresets[presetName]
	if !ok {
		return preset{}, fmt.Errorf("unknown preset %q, valid presets: %v", presetName, sortedPresetNames)
	}
	if where != "" {
		p.Where = p.Where + " and " + where
	}
	return p, nil
}

func newPresetsCmd() *cli.Command {
	return &cli.Command{
		Name:  "presets",
		Usage: "List available search preset filters",
		UsageText: `# List all presets in text format
  tp entity presets

  # List presets as JSON
  tp entity presets --output json`,
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
				names := sortedPresetNames
				presets := make([]jsonPreset, len(names))
				for i, name := range names {
					p := searchPresets[name]
					presets[i] = jsonPreset(p)
				}
				return output.PrintJSON(os.Stdout, map[string]any{
					"presets": presets,
				})
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(tw, "NAME\tDESCRIPTION\tWHERE\n")
			for _, name := range sortedPresetNames {
				p := searchPresets[name]
				fmt.Fprintf(tw, "%s\t%s\t%s\n", name, p.Description, p.Where)
			}
			return tw.Flush()
		},
	}
}
