package cheatsheet

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
)

const markdownCheatsheet = `# tp CLI — Quick Reference

## Commands

### tp query <Type> [flags]
Query entities using v2 API with powerful filtering and projections.
  -s, --select    Fields to return (e.g., 'id,name,entityState.name as state')
  -w, --where     Filter expression
  --order         Sort (e.g., 'createDate desc')
  -t, --take      Max results (default 25, max 1000)
  --skip          Skip N results
  --dry-run       Show URL without executing

### tp entity search --type <Type> [flags]
Search entities using v1 API.
  --type          Entity type (required)
  --where         Filter expression (v1 syntax)
  --include       Fields to include
  --take          Max results
  --order-by      Sort fields
  --preset        Use a preset filter

### tp entity get --type <Type> --id <ID>
Get a single entity by ID.

### tp entity create --type <Type> --name <Name> --project-id <ID>
Create a new entity.

### tp entity update --type <Type> --id <ID> [field flags]
Update an entity (--name, --description, --state-id, --assigned-user-id).

### tp entity presets
List available search presets.

### tp entity comment list --entity-id <ID>
List comments on an entity.

### tp entity comment add --entity-id <ID> --body <text>
Add a comment (auto-markdown, @mention resolution).

### tp entity comment delete --id <ID>
Delete a comment by ID.

### tp inspect types
List all available entity types.

### tp inspect properties --type <Type>
List properties of an entity type.

### tp inspect details --type <Type> --property <Name>
Get detailed info about a property.

### tp inspect discover
Discover available entity types.

### tp api [METHOD] <path> [--body JSON]
Make raw API requests.

### tp config get|set|list|path
Manage configuration.

## Entity Types
Common: UserStory, Bug, Task, Feature, Epic, Request
Cross-type: Assignable (all work items), General (everything)
Other: Project, Team, Iteration, TeamIteration, Release, Program, Comment, Time, Assignment, Relation, EntityState, User

## v2 Query Syntax Reference

### Select
  {id,name}                              — basic fields
  entityState.name as state              — dot-path MUST use 'as' alias
  tasks.count as taskCount               — collection count
  tasks.select({id,name}) as taskList    — nested projection
  tasks.where(cond).count as cnt         — filtered count
  tasks.sum(effort) as total             — aggregation (sum/avg/min/max)

### Where Operators
  ==, !=, >, <, >=, <=                   — comparison
  and, or, not(...)                      — logical
  entityState.name in ["Open","Done"]    — membership
  name.contains("text")                  — substring
  name.startsWith("prefix")             — prefix match
  name.toLower().contains("text")        — case-insensitive
  field==null, field!=null               — null check (NOT 'is null')
  entityState.isFinal==true              — done states
  entityState.isInitial==true            — open states
  assignments.any(generalUser.id==123)   — collection predicate

### Date Functions
  Today                                  — current date
  Today.AddDays(-7)                      — relative days
  Today.AddMonths(-1)                    — relative months
  Today.AddHours(-24)                    — relative hours
  NOTE: 'Today - 7' does NOT work, use AddDays(-7)

### OrderBy
  createDate desc                        — descending
  priority.importance desc,name asc      — multiple fields

## Search Presets
  open, inProgress, done, unassigned, highPriority,
  createdToday, modifiedToday, createdThisWeek, modifiedThisWeek,
  createdLastWeek, modifiedLastWeek, highPriorityUnassigned

## Common Query Examples

  # All open bugs
  tp query Bug -w 'entityState.isFinal!=true' -s 'id,name,priority.name as priority'

  # Sprint status
  tp query Assignable -s 'id,name,entityType.name as type,entityState.name as state' -w 'teamIteration!=null'

  # Feature progress
  tp query Feature -s 'id,name,userStories.count as total,userStories.where(entityState.isFinal==true).count as done'

  # Recently modified
  tp query Assignable -s 'id,name,entityType.name as type' -w 'modifyDate>=Today.AddDays(-7)' --order 'modifyDate desc'

  # Cross-type text search
  tp query Assignable -s 'id,name,entityType.name as type' -w 'name.toLower().contains("keyword")'

  # Items assigned to someone
  tp query UserStory -s 'id,name' -w 'assignments.any(generalUser.firstName=="John")'

  # List comments
  tp entity comment list --entity-id 342236

  # Add a comment with @mention
  tp entity comment add --entity-id 342236 --body "Hey @timo, looks good"

  # Raw API call
  tp api GET '/api/v1/UserStorys?take=5'
  tp api POST '/api/v1/Comments' --body '{"General":{"Id":123},"Description":"Hello"}'
`

// NewCmd creates the cheatsheet command.
func NewCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "cheatsheet",
		Usage: "Print a compact CLI reference (useful for LLM agents)",
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmdutil.IsJSON(cmd) {
				return output.PrintJSON(os.Stdout, jsonCheatsheet())
			}
			fmt.Fprint(os.Stdout, markdownCheatsheet)
			return nil
		},
	}
}

func jsonCheatsheet() map[string]any {
	return map[string]any{
		"commands": []map[string]any{
			{
				"name":  "tp query",
				"usage": "Query entities via v2 API",
				"flags": []map[string]string{
					{"name": "-s, --select", "usage": "Fields to return"},
					{"name": "-w, --where", "usage": "Filter expression"},
					{"name": "--order", "usage": "Sort expression"},
					{"name": "-t, --take", "usage": "Max results (default 25, max 1000)"},
					{"name": "--skip", "usage": "Skip N results"},
					{"name": "--dry-run", "usage": "Show URL without executing"},
				},
			},
			{
				"name":  "tp entity search",
				"usage": "Search entities via v1 API",
				"flags": []map[string]string{
					{"name": "--type", "usage": "Entity type (required)"},
					{"name": "--where", "usage": "Filter expression (v1 syntax)"},
					{"name": "--include", "usage": "Fields to include"},
					{"name": "--take", "usage": "Max results"},
					{"name": "--order-by", "usage": "Sort fields"},
					{"name": "--preset", "usage": "Use a preset filter"},
				},
			},
			{
				"name":  "tp entity get",
				"usage": "Get a single entity by ID",
				"flags": []map[string]string{
					{"name": "--type", "usage": "Entity type (required)"},
					{"name": "--id", "usage": "Entity ID (required)"},
				},
			},
			{
				"name":  "tp entity create",
				"usage": "Create a new entity",
				"flags": []map[string]string{
					{"name": "--type", "usage": "Entity type (required)"},
					{"name": "--name", "usage": "Entity name (required)"},
					{"name": "--project-id", "usage": "Project ID (required)"},
				},
			},
			{
				"name":  "tp entity update",
				"usage": "Update an entity",
				"flags": []map[string]string{
					{"name": "--type", "usage": "Entity type (required)"},
					{"name": "--id", "usage": "Entity ID (required)"},
					{"name": "--name", "usage": "New name"},
					{"name": "--description", "usage": "New description"},
					{"name": "--state-id", "usage": "New state ID"},
					{"name": "--assigned-user-id", "usage": "Assigned user ID"},
				},
			},
			{
				"name":  "tp entity presets",
				"usage": "List available search presets",
			},
			{
				"name":  "tp entity comment list",
				"usage": "List comments on an entity",
				"flags": []map[string]string{
					{"name": "--entity-id", "usage": "Entity ID (required)"},
				},
			},
			{
				"name":  "tp entity comment add",
				"usage": "Add a comment (auto-markdown, @mention resolution)",
				"flags": []map[string]string{
					{"name": "--entity-id", "usage": "Entity ID (required)"},
					{"name": "--body", "usage": "Comment text (required)"},
				},
			},
			{
				"name":  "tp entity comment delete",
				"usage": "Delete a comment by ID",
				"flags": []map[string]string{
					{"name": "--id", "usage": "Comment ID (required)"},
				},
			},
			{
				"name":  "tp inspect types",
				"usage": "List all available entity types",
			},
			{
				"name":  "tp inspect properties",
				"usage": "List properties of an entity type",
				"flags": []map[string]string{
					{"name": "--type", "usage": "Entity type (required)"},
				},
			},
			{
				"name":  "tp inspect details",
				"usage": "Get detailed info about a property",
				"flags": []map[string]string{
					{"name": "--type", "usage": "Entity type (required)"},
					{"name": "--property", "usage": "Property name (required)"},
				},
			},
			{
				"name":  "tp inspect discover",
				"usage": "Discover available entity types",
			},
			{
				"name":  "tp api",
				"usage": "Make raw API requests",
				"flags": []map[string]string{
					{"name": "--body", "usage": "Request body (JSON string)"},
				},
			},
			{
				"name":  "tp config get|set|list|path",
				"usage": "Manage configuration",
			},
		},
		"entityTypes": []string{
			"UserStory", "Bug", "Task", "Feature", "Epic", "Request",
			"Assignable", "General",
			"Project", "Team", "Iteration", "TeamIteration", "Release",
			"Program", "Comment", "Time", "Assignment", "Relation",
			"EntityState", "User",
		},
		"v2Syntax": map[string]any{
			"select": []string{
				"{id,name} — basic fields",
				"entityState.name as state — dot-path with alias",
				"tasks.count as taskCount — collection count",
				"tasks.select({id,name}) as taskList — nested projection",
				"tasks.where(cond).count as cnt — filtered count",
				"tasks.sum(effort) as total — aggregation (sum/avg/min/max)",
			},
			"where": []string{
				"==, !=, >, <, >=, <= — comparison",
				"and, or, not(...) — logical",
				"entityState.name in [\"Open\",\"Done\"] — membership",
				"name.contains(\"text\") — substring",
				"name.startsWith(\"prefix\") — prefix match",
				"name.toLower().contains(\"text\") — case-insensitive",
				"field==null, field!=null — null check (NOT 'is null')",
				"entityState.isFinal==true — done states",
				"entityState.isInitial==true — open states",
				"assignments.any(generalUser.id==123) — collection predicate",
			},
			"dateFunctions": []string{
				"Today — current date",
				"Today.AddDays(-7) — relative days",
				"Today.AddMonths(-1) — relative months",
				"Today.AddHours(-24) — relative hours",
				"NOTE: 'Today - 7' does NOT work, use AddDays(-7)",
			},
			"orderBy": []string{
				"createDate desc — descending",
				"priority.importance desc,name asc — multiple fields",
			},
		},
		"presets": []string{
			"open", "inProgress", "done", "unassigned", "highPriority",
			"createdToday", "modifiedToday", "createdThisWeek", "modifiedThisWeek",
			"createdLastWeek", "modifiedLastWeek", "highPriorityUnassigned",
		},
		"examples": []map[string]string{
			{"description": "All open bugs", "command": "tp query Bug -w 'entityState.isFinal!=true' -s 'id,name,priority.name as priority'"},
			{"description": "Sprint status", "command": "tp query Assignable -s 'id,name,entityType.name as type,entityState.name as state' -w 'teamIteration!=null'"},
			{"description": "Feature progress", "command": "tp query Feature -s 'id,name,userStories.count as total,userStories.where(entityState.isFinal==true).count as done'"},
			{"description": "Recently modified", "command": "tp query Assignable -s 'id,name,entityType.name as type' -w 'modifyDate>=Today.AddDays(-7)' --order 'modifyDate desc'"},
			{"description": "Cross-type text search", "command": "tp query Assignable -s 'id,name,entityType.name as type' -w 'name.toLower().contains(\"keyword\")'"},
			{"description": "Items assigned to someone", "command": "tp query UserStory -s 'id,name' -w 'assignments.any(generalUser.firstName==\"John\")'"},
			{"description": "List comments", "command": "tp entity comment list --entity-id 342236"},
			{"description": "Add a comment with @mention", "command": "tp entity comment add --entity-id 342236 --body \"Hey @timo, looks good\""},
			{"description": "Raw API GET", "command": "tp api GET '/api/v1/UserStorys?take=5'"},
			{"description": "Raw API POST", "command": "tp api POST '/api/v1/Comments' --body '{\"General\":{\"Id\":123},\"Description\":\"Hello\"}'"},
		},
	}
}
