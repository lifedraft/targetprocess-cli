package resolve

import "strings"

// knownTypes maps lowercase entity type names to their canonical form.
var knownTypes = map[string]string{
	"userstory":      "UserStory",
	"bug":            "Bug",
	"task":           "Task",
	"feature":        "Feature",
	"epic":           "Epic",
	"request":        "Request",
	"assignable":     "Assignable",
	"project":        "Project",
	"team":           "Team",
	"assignment":     "Assignment",
	"relation":       "Relation",
	"comment":        "Comment",
	"time":           "Time",
	"iteration":      "Iteration",
	"teamiteration":  "TeamIteration",
	"release":        "Release",
	"testcase":       "TestCase",
	"testplan":       "TestPlan",
	"testplanrun":    "TestPlanRun",
	"impediment":     "Impediment",
	"general":        "General",
	"generaluser":    "GeneralUser",
	"role":           "Role",
	"entitystate":    "EntityState",
	"priority":       "Priority",
	"severity":       "Severity",
	"program":        "Program",
	"portfolioepic":  "PortfolioEpic",
	"solution":       "Solution",
	"teamassignment": "TeamAssignment",
}

// plurals maps lowercase plural forms to their lowercase singular form.
var plurals = map[string]string{
	"userstories":     "userstory",
	"stories":         "userstory",
	"bugs":            "bug",
	"tasks":           "task",
	"features":        "feature",
	"epics":           "epic",
	"requests":        "request",
	"projects":        "project",
	"teams":           "team",
	"assignments":     "assignment",
	"relations":       "relation",
	"comments":        "comment",
	"iterations":      "iteration",
	"teamiterations":  "teamiteration",
	"releases":        "release",
	"testcases":       "testcase",
	"testplans":       "testplan",
	"testplanruns":    "testplanrun",
	"impediments":     "impediment",
	"programs":        "program",
	"portfolioepics":  "portfolioepic",
	"solutions":       "solution",
	"teamassignments": "teamassignment",
}

// aliases maps common shorthand names to their lowercase canonical form.
var aliases = map[string]string{
	"story":      "userstory",
	"us":         "userstory",
	"issue":      "bug",
	"defect":     "bug",
	"iteration":  "iteration",
	"sprint":     "iteration",
	"test":       "testcase",
	"assignable": "assignable",
}

// EntityType resolves a user-provided entity type string to its canonical
// Targetprocess API form. It handles:
//   - Case-insensitive matching: "userstory" → "UserStory"
//   - Plural stripping: "UserStories" → "UserStory", "Bugs" → "Bug"
//   - Aliases: "story" → "UserStory", "us" → "UserStory"
//
// Unknown types pass through unchanged so the API can validate them.
func EntityType(input string) string {
	lower := strings.ToLower(input)

	// 1. Direct case-insensitive match
	if canonical, ok := knownTypes[lower]; ok {
		return canonical
	}

	// 2. Plural → singular
	if singular, ok := plurals[lower]; ok {
		if canonical, ok := knownTypes[singular]; ok {
			return canonical
		}
	}

	// 3. Alias
	if target, ok := aliases[lower]; ok {
		if canonical, ok := knownTypes[target]; ok {
			return canonical
		}
	}

	// Unknown type: pass through unchanged
	return input
}
