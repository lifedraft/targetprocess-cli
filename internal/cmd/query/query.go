package query

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/api"
	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
)

// NewCmd creates the "query" command for v2 API queries.
func NewCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:      "query",
		Usage:     "Query Targetprocess entities using API v2",
		ArgsUsage: "<EntityType>[/<id>]",
		UsageText: `# Search across all work item types
  tp query Assignable -s 'id,name,entityType.name as type,entityState.name as state' -w 'entityState.isFinal!=true' --take 20

  # Get a single entity with deep data
  tp query UserStory/342236 -s 'id,name,entityState.name as state,feature.name as feature,tasks.count as taskCount'

  # Sprint status report
  tp query UserStory -s 'id,name,entityState.name as state,effort' -w 'teamIteration!=null' --order 'entityState.name'

  # Feature progress with rollup counts
  tp query Feature -s 'id,name,userStories.count as total,userStories.where(entityState.isFinal==true).count as done' -w 'project.name=="Mobile App"'

  # Find items by text search
  tp query Assignable -s 'id,name,entityType.name as type' -w 'name.toLower().contains("login")' --order 'modifyDate desc'

  # Dry run to inspect the URL
  tp query Bug -w 'entityState.name=="Open"' --dry-run

  # Items created in last 7 days
  tp query UserStory -s 'id,name,createDate' -w 'createDate>=Today.AddDays(-7)' --order 'createDate desc'

  # Team workload via assignments
  tp query Assignment -s 'generalUser.firstName as person,assignable.name as item,assignable.effort as effort' -w 'assignable.entityState.isFinal!=true'`,
		Description: `Query Targetprocess using API v2's powerful query language.

Entity types: UserStory, Bug, Task, Feature, Epic, Request, Assignable (all types), Project, Team, Assignment, Relation, Comment, Time

Select syntax: field, ref.field as alias, collection.count, collection.where(cond).select({fields})
Where operators: ==, !=, >, <, >=, <=, and, or, in [...], .contains(), .startsWith(), .toLower()
Date functions: Today, Today.AddDays(-N), Today.AddMonths(-N)
Null checks: field==null, field!=null
State helpers: entityState.isFinal==true, entityState.isInitial==true`,
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.StringFlag{
				Name:    "select",
				Aliases: []string{"s"},
				Usage:   "Select expression (e.g., 'id,name,entityState.name as state')",
			},
			&cli.StringFlag{
				Name:    "where",
				Aliases: []string{"w"},
				Usage:   "Where filter expression",
			},
			&cli.StringFlag{
				Name:  "order",
				Usage: "OrderBy expression (e.g., 'createDate desc')",
			},
			&cli.IntFlag{
				Name:    "take",
				Aliases: []string{"t"},
				Value:   25,
				Usage:   "Max number of results to return",
			},
			&cli.IntFlag{
				Name:  "skip",
				Value: 0,
				Usage: "Number of results to skip",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Show the URL that would be called without executing",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) == 0 {
				return errors.New("entity type is required; usage: tp query <EntityType>[/<id>]")
			}

			entityType, entityID, err := parseEntityArg(args[0])
			if err != nil {
				return err
			}

			client, err := f.Client()
			if err != nil {
				return err
			}

			selectExpr := cmd.String("select")

			// Warn about dot-paths missing 'as' aliases (silently dropped by API)
			if warn := api.WarnSelectDotPaths(selectExpr); warn != "" {
				fmt.Fprint(os.Stderr, warn)
			}

			// Single entity by ID
			if entityID > 0 {
				if cmd.Bool("dry-run") {
					fmt.Fprintln(os.Stdout, client.BuildV2EntityURL(entityType, entityID, selectExpr))
					return nil
				}

				var data []byte
				data, err = client.QueryV2Entity(ctx, entityType, entityID, selectExpr)
				if err != nil {
					path := fmt.Sprintf("/api/v2/%s/%d", entityType, entityID)
					err = api.EnhanceError(err, path, map[string]string{"select": selectExpr})
					return fmt.Errorf("query failed: %w", err)
				}

				return printResponse(cmd, data)
			}

			// Collection query
			params := api.V2Params{
				Where:   cmd.String("where"),
				Select:  selectExpr,
				OrderBy: cmd.String("order"),
				Take:    cmd.Int("take"),
				Skip:    cmd.Int("skip"),
			}

			if cmd.Bool("dry-run") {
				fmt.Fprintln(os.Stdout, client.BuildV2URL(entityType, params))
				return nil
			}

			data, err := client.QueryV2(ctx, entityType, params)
			if err != nil {
				path := fmt.Sprintf("/api/v2/%s", entityType)
				err = api.EnhanceError(err, path, map[string]string{
					"where":   params.Where,
					"select":  params.Select,
					"orderBy": params.OrderBy,
				})
				return fmt.Errorf("query failed: %w", err)
			}

			return printResponse(cmd, data)
		},
	}
}

// parseEntityArg splits "EntityType" or "EntityType/123" into parts.
func parseEntityArg(arg string) (entityType string, id int, err error) {
	parts := strings.SplitN(arg, "/", 2)
	entityType = parts[0]
	if entityType == "" {
		return "", 0, errors.New("entity type cannot be empty")
	}
	if len(parts) == 2 {
		id, err = strconv.Atoi(parts[1])
		if err != nil {
			return "", 0, fmt.Errorf("invalid entity ID %q: must be an integer", parts[1])
		}
		return entityType, id, nil
	}
	return entityType, 0, nil
}

// printResponse handles output for any v2 response (single entity or collection).
func printResponse(cmd *cli.Command, data []byte) error {
	if cmdutil.IsJSON(cmd) {
		var parsed any
		if err := json.Unmarshal(data, &parsed); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
		return output.PrintJSON(os.Stdout, parsed)
	}

	// Try to parse as collection first
	var resp struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(data, &resp); err == nil && resp.Items != nil {
		if len(resp.Items) == 0 {
			fmt.Fprintln(os.Stdout, "No results found.")
			return nil
		}
		printDynamicTable(resp.Items)
		return nil
	}

	// Single entity
	var entity map[string]any
	if err := json.Unmarshal(data, &entity); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}
	output.PrintEntity(os.Stdout, entity)
	return nil
}

// printDynamicTable prints items as a table, deriving columns from the data.
func printDynamicTable(items []map[string]any) {
	colSet := make(map[string]bool)
	var cols []string
	for _, item := range items {
		for key := range item {
			if key == "resourceType" {
				continue
			}
			if !colSet[key] {
				colSet[key] = true
				cols = append(cols, key)
			}
		}
	}
	sort.Strings(cols)

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = strings.ToUpper(c)
	}
	fmt.Fprintln(tw, strings.Join(headers, "\t"))

	for _, item := range items {
		vals := make([]string, len(cols))
		for i, col := range cols {
			vals[i] = formatValue(item[col])
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}
	tw.Flush()
}

// formatValue converts a value to a display string.
func formatValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case map[string]any:
		if name, ok := val["name"]; ok {
			return fmt.Sprintf("%v", name)
		}
		if name, ok := val["Name"]; ok {
			return fmt.Sprintf("%v", name)
		}
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	case []any:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = formatValue(item)
		}
		return strings.Join(parts, ", ")
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return fmt.Sprintf("%g", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
