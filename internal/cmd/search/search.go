package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/api"
	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
	"github.com/lifedraft/targetprocess-cli/internal/resolve"
)

// NewCmd creates the "search" command.
func NewCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:      "search",
		Usage:     "Search for entities using Targetprocess API v2",
		ArgsUsage: "<type>",
		UsageText: `# Search open user stories
  tp search UserStory -w 'entityState.isFinal!=true' -s 'id,name,entityState.name as state'

  # Cross-type search
  tp search Assignable -w 'name.toLower().contains("login")' -s 'id,name,entityType.name as type'

  # Use a preset
  tp search UserStory --preset open

  # With sorting
  tp search Bug -w 'priority.name=="High"' --order-by 'createDate desc' --take 50

  # Recently modified items
  tp search Assignable --preset recentActivity`,
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.StringFlag{
				Name:    "where",
				Aliases: []string{"w"},
				Usage:   `Filter expression using v2 syntax (e.g. 'entityState.isFinal!=true', 'name.contains("login")')`,
			},
			&cli.StringFlag{
				Name:  "preset",
				Usage: "Use a preset filter (run 'tp presets' to list available presets)",
			},
			&cli.StringFlag{
				Name:    "select",
				Aliases: []string{"s"},
				Usage:   `Fields to return, comma-separated (e.g. 'id,name,entityState.name as state')`,
			},
			&cli.IntFlag{
				Name:    "take",
				Aliases: []string{"t"},
				Value:   25,
				Usage:   "Max number of results to return (max 1000)",
			},
			&cli.StringFlag{
				Name:  "order-by",
				Usage: "Sort expression (e.g. 'createDate desc')",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) == 0 {
				return errors.New("entity type is required; usage: tp search <type> [flags]")
			}

			entityType := resolve.EntityType(args[0])
			if vErr := api.ValidateEntityType(entityType); vErr != nil {
				return vErr
			}

			client, err := f.Client()
			if err != nil {
				return err
			}

			where := cmd.String("where")
			selectExpr := cmd.String("select")
			take := cmd.Int("take")
			orderBy := cmd.String("order-by")

			// Apply preset if specified
			if presetName := cmd.String("preset"); presetName != "" {
				var p Preset
				p, err = ApplyPreset(presetName, where)
				if err != nil {
					return err
				}
				where = p.Where
				if selectExpr == "" && p.Select != "" {
					selectExpr = p.Select
				}
				if orderBy == "" && p.OrderBy != "" {
					orderBy = p.OrderBy
				}
			}

			if take < 0 || take > 1000 {
				return fmt.Errorf("take must be between 0 and 1000, got %d", take)
			}

			// Warn about dot-paths missing 'as' aliases (silently dropped by API)
			if warn := api.WarnSelectDotPaths(selectExpr); warn != "" {
				fmt.Fprint(os.Stderr, warn)
			}

			params := api.V2Params{
				Where:   where,
				Select:  selectExpr,
				OrderBy: orderBy,
				Take:    take,
			}

			data, err := client.QueryV2(ctx, entityType, params)
			if err != nil {
				path := fmt.Sprintf("/api/v2/%s", entityType)
				err = api.EnhanceError(err, path, map[string]string{
					"where":   params.Where,
					"select":  params.Select,
					"orderBy": params.OrderBy,
				})
				return fmt.Errorf("search failed: %w", err)
			}

			// Parse v2 response: {"items": [...], "next": "..."}
			var resp struct {
				Items []api.Entity `json:"items"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing v2 response: %w", err)
			}

			if cmdutil.IsJSON(cmd) {
				return output.PrintJSON(os.Stdout, map[string]any{
					"items": resp.Items,
					"count": len(resp.Items),
				})
			}

			printV2EntityTable(os.Stdout, resp.Items)
			return nil
		},
	}
}

// printV2EntityTable prints entities from the v2 API as a table.
func printV2EntityTable(w io.Writer, entities []api.Entity) {
	if len(entities) == 0 {
		fmt.Fprintln(w, "No results found.")
		return
	}

	cols := detectColumns(entities[0])

	tw := output.NewTabWriter(w)
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = strings.ToUpper(c.label)
	}
	fmt.Fprintln(tw, strings.Join(headers, "\t"))

	for _, e := range entities {
		vals := make([]string, len(cols))
		for i, c := range cols {
			vals[i] = fmt.Sprintf("%v", c.extract(e))
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}
	tw.Flush()
}

type column struct {
	label   string
	extract func(api.Entity) any
}

func getField(e api.Entity, camel, pascal string) any {
	if v, ok := e[camel]; ok {
		return v
	}
	if v, ok := e[pascal]; ok {
		return v
	}
	return ""
}

func getNestedName(e api.Entity, aliasKey, nestedObjKey string) any {
	if v, ok := e[aliasKey]; ok {
		return v
	}
	if nestedObjKey == "" {
		return ""
	}
	if obj, ok := e[nestedObjKey].(map[string]any); ok {
		if n, ok := obj["Name"]; ok {
			return n
		}
		if n, ok := obj["name"]; ok {
			return n
		}
	}
	camelKey := strings.ToLower(nestedObjKey[:1]) + nestedObjKey[1:]
	if obj, ok := e[camelKey].(map[string]any); ok {
		if n, ok := obj["name"]; ok {
			return n
		}
		if n, ok := obj["Name"]; ok {
			return n
		}
	}
	return ""
}

func detectColumns(sample api.Entity) []column {
	var cols []column

	if _, ok := sample["id"]; ok {
		cols = append(cols, column{label: "id", extract: func(e api.Entity) any { return getField(e, "id", "Id") }})
	} else if _, ok := sample["Id"]; ok {
		cols = append(cols, column{label: "id", extract: func(e api.Entity) any { return getField(e, "id", "Id") }})
	}

	if _, ok := sample["name"]; ok {
		cols = append(cols, column{label: "name", extract: func(e api.Entity) any { return getField(e, "name", "Name") }})
	} else if _, ok := sample["Name"]; ok {
		cols = append(cols, column{label: "name", extract: func(e api.Entity) any { return getField(e, "name", "Name") }})
	}

	if _, ok := sample["type"]; ok {
		cols = append(cols, column{label: "type", extract: func(e api.Entity) any { return getField(e, "type", "ResourceType") }})
	} else if _, ok := sample["ResourceType"]; ok {
		cols = append(cols, column{label: "type", extract: func(e api.Entity) any { return getField(e, "ResourceType", "ResourceType") }})
	} else if _, ok := sample["entityType"]; ok {
		cols = append(cols, column{label: "type", extract: func(e api.Entity) any {
			return getNestedName(e, "type", "EntityType")
		}})
	}

	if _, ok := sample["state"]; ok {
		cols = append(cols, column{label: "state", extract: func(e api.Entity) any { return getField(e, "state", "state") }})
	} else if _, ok := sample["entityState"]; ok {
		cols = append(cols, column{label: "state", extract: func(e api.Entity) any {
			return getNestedName(e, "state", "EntityState")
		}})
	} else if _, ok := sample["EntityState"]; ok {
		cols = append(cols, column{label: "state", extract: func(e api.Entity) any {
			return getNestedName(e, "state", "EntityState")
		}})
	}

	knownKeys := map[string]bool{
		"id": true, "Id": true, "name": true, "Name": true,
		"type": true, "ResourceType": true, "entityType": true, "EntityType": true,
		"state": true, "entityState": true, "EntityState": true,
	}

	var extraKeys []string
	for key := range sample {
		if !knownKeys[key] {
			extraKeys = append(extraKeys, key)
		}
	}
	sort.Strings(extraKeys)

	for _, k := range extraKeys {
		key := k
		cols = append(cols, column{label: key, extract: func(e api.Entity) any {
			v := e[key]
			if v == nil {
				return ""
			}
			if obj, ok := v.(map[string]any); ok {
				if n, ok := obj["name"]; ok {
					return n
				}
				if n, ok := obj["Name"]; ok {
					return n
				}
			}
			return v
		}})
	}

	if len(cols) == 0 {
		var allKeys []string
		for key := range sample {
			allKeys = append(allKeys, key)
		}
		sort.Strings(allKeys)
		for _, k := range allKeys {
			key := k
			cols = append(cols, column{label: key, extract: func(e api.Entity) any { return e[key] }})
		}
	}

	return cols
}
