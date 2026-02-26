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

### tp show <id> [flags]
Show a single entity by ID (auto-detects type).
  --type          Entity type (skip auto-detection)
  --include       Related data to include (e.g. Project,Team)
  -o, --output    Output format: text, json

### tp search <type> [flags]
Search entities using v2 API.
  -w, --where     Filter expression (e.g. 'entityState.isFinal!=true')
  -s, --select    Fields to return (e.g. 'id,name,entityState.name as state')
  --preset        Use a preset filter (run 'tp presets' to list)
  -t, --take      Max results (default 25, max 1000)
  --order-by      Sort expression (e.g. 'createDate desc')

### tp create <type> <name> --project-id <ID>
Create a new entity.
  --project-id    Project ID (required)
  --description   Entity description
  --team-id       Team ID
  --assigned-user-id  Assigned user ID

### tp update <id> [flags]
Update an entity (auto-detects type).
  --type          Entity type (skip auto-detection)
  --name          New name
  --description   New description
  --state-id      New entity state ID
  --assigned-user-id  New assigned user ID

### tp comment list <entity-id>
List comments on an entity.

### tp comment add <entity-id> <body>
Add a comment (auto-markdown, @mention resolution).

### tp comment delete <comment-id>
Delete a comment by ID.

### tp presets
List available search presets.

### tp query <Type>[/<id>] [flags]
Query entities using v2 API with powerful filtering and projections.
  -s, --select    Fields to return (e.g., 'id,name,entityState.name as state')
  -w, --where     Filter expression
  --order         Sort (e.g., 'createDate desc')
  -t, --take      Max results (default 25, max 1000)
  --skip          Skip N results
  --dry-run       Show URL without executing

### tp inspect types|properties|details|discover
Inspect Targetprocess API metadata.

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

## Common Examples

  # Show an entity by ID
  tp show 341079

  # All open bugs
  tp search Bug -w 'entityState.isFinal!=true' -s 'id,name,priority.name as priority'

  # Cross-type search
  tp search Assignable -s 'id,name,entityType.name as type' -w 'name.toLower().contains("keyword")'

  # Use a preset
  tp search UserStory --preset open

  # Create a story
  tp create UserStory "Implement login" --project-id 42

  # Update an entity
  tp update 12345 --name "New title"

  # List comments
  tp comment list 342236

  # Add a comment with @mention
  tp comment add 342236 "Hey @timo, looks good"

  # Advanced query with projections
  tp query Feature -s 'id,name,userStories.count as total,userStories.where(entityState.isFinal==true).count as done'

  # Raw API call
  tp api GET '/api/v1/UserStorys?take=5'
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
				"name":  "tp show",
				"usage": "Show entity by ID (auto-detects type)",
				"args":  "<id>",
				"flags": []map[string]string{
					{"name": "--type", "usage": "Entity type (skip auto-detection)"},
					{"name": "--include", "usage": "Related data to include"},
				},
			},
			{
				"name":  "tp search",
				"usage": "Search entities using v2 API",
				"args":  "<type>",
				"flags": []map[string]string{
					{"name": "-w, --where", "usage": "Filter expression"},
					{"name": "-s, --select", "usage": "Fields to return"},
					{"name": "--preset", "usage": "Use a preset filter"},
					{"name": "-t, --take", "usage": "Max results (default 25, max 1000)"},
					{"name": "--order-by", "usage": "Sort expression"},
				},
			},
			{
				"name":  "tp create",
				"usage": "Create a new entity",
				"args":  "<type> <name>",
				"flags": []map[string]string{
					{"name": "--project-id", "usage": "Project ID (required)"},
					{"name": "--description", "usage": "Entity description"},
					{"name": "--team-id", "usage": "Team ID"},
					{"name": "--assigned-user-id", "usage": "Assigned user ID"},
				},
			},
			{
				"name":  "tp update",
				"usage": "Update entity (auto-detects type)",
				"args":  "<id>",
				"flags": []map[string]string{
					{"name": "--type", "usage": "Entity type (skip auto-detection)"},
					{"name": "--name", "usage": "New name"},
					{"name": "--description", "usage": "New description"},
					{"name": "--state-id", "usage": "New state ID"},
					{"name": "--assigned-user-id", "usage": "Assigned user ID"},
				},
			},
			{
				"name":  "tp comment list",
				"usage": "List comments on an entity",
				"args":  "<entity-id>",
			},
			{
				"name":  "tp comment add",
				"usage": "Add a comment (auto-markdown, @mention resolution)",
				"args":  "<entity-id> <body>",
			},
			{
				"name":  "tp comment delete",
				"usage": "Delete a comment by ID",
				"args":  "<comment-id>",
			},
			{
				"name":  "tp presets",
				"usage": "List available search presets",
			},
			{
				"name":  "tp query",
				"usage": "Query entities via v2 API",
				"args":  "<Type>[/<id>]",
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
				"name":  "tp inspect",
				"usage": "Inspect API metadata (types, properties, details, discover)",
			},
			{
				"name":  "tp api",
				"usage": "Make raw API requests",
				"flags": []map[string]string{
					{"name": "--body", "usage": "Request body (JSON string)"},
				},
			},
			{
				"name":  "tp config",
				"usage": "Manage configuration (get, set, list, path)",
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
			{"description": "Show entity by ID", "command": "tp show 341079"},
			{"description": "All open bugs", "command": "tp search Bug -w 'entityState.isFinal!=true' -s 'id,name,priority.name as priority'"},
			{"description": "Cross-type search", "command": "tp search Assignable -s 'id,name,entityType.name as type' -w 'name.toLower().contains(\"keyword\")'"},
			{"description": "Use a preset", "command": "tp search UserStory --preset open"},
			{"description": "Create a story", "command": "tp create UserStory \"Implement login\" --project-id 42"},
			{"description": "Update an entity", "command": "tp update 12345 --name \"New title\""},
			{"description": "List comments", "command": "tp comment list 342236"},
			{"description": "Add a comment with @mention", "command": "tp comment add 342236 \"Hey @timo, looks good\""},
			{"description": "Feature progress", "command": "tp query Feature -s 'id,name,userStories.count as total,userStories.where(entityState.isFinal==true).count as done'"},
			{"description": "Raw API GET", "command": "tp api GET '/api/v1/UserStorys?take=5'"},
		},
	}
}
