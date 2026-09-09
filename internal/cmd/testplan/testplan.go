package testplan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
)

// NewCmd creates the "testplan" command.
func NewCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "testplan",
		Usage: "Manage test plans",
		UsageText: `# Create a test plan for a story/bug (with a test case, linked to a release plan)
  tp testplan create 363376 --release-plan 365157

  # Add child test plans to a parent test plan
  tp testplan add 365157 347067 360449 361777

  # List child test plans of a test plan
  tp testplan list-children 365157`,
		Commands: []*cli.Command{
			newCreateCmd(f),
			newAddChildCmd(f),
			newListChildrenCmd(f),
		},
	}
}

// htmlToMarkdown converts an HTML snippet (as returned by TP) into clean
// markdown text suitable for a test case description.
func htmlToMarkdown(s string) string {
	// Remove <!--markdown--> prefix used by TP.
	s = strings.TrimPrefix(s, "<!--markdown-->")

	// Decode all HTML entities first so href/text values are readable.
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00a0", " ")

	// Convert <a href="url">text</a>:
	//   same text as href → bare URL
	//   different text    → [text](url)
	linkRe := regexp.MustCompile(`(?is)<a[^>]*\shref="([^"]*)"[^>]*>(.*?)</a>`)
	strTags := regexp.MustCompile(`<[^>]+>`)
	s = linkRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := linkRe.FindStringSubmatch(m)
		if len(sub) < 3 {
			return m
		}
		href := strings.TrimSpace(sub[1])
		text := strings.TrimSpace(strTags.ReplaceAllString(sub[2], ""))
		if text == "" || text == href {
			return href
		}
		return fmt.Sprintf("[%s](%s)", text, href)
	})

	// Convert headings.
	for i := 6; i >= 1; i-- {
		hRe := regexp.MustCompile(fmt.Sprintf(`(?is)<h%d[^>]*>(.*?)</h%d>`, i, i))
		prefix := strings.Repeat("#", i)
		s = hRe.ReplaceAllStringFunc(s, func(m string) string {
			sub := hRe.FindStringSubmatch(m)
			text := strings.TrimSpace(strTags.ReplaceAllString(sub[1], ""))
			return fmt.Sprintf("\n%s %s\n", prefix, text)
		})
	}

	// Convert <hr> to a divider.
	s = regexp.MustCompile(`(?i)<hr[^>]*/?>`).ReplaceAllString(s, "\n")

	// Convert <ol> list items to numbered markdown.
	olRe := regexp.MustCompile(`(?is)<ol[^>]*>(.*?)</ol>`)
	liRe := regexp.MustCompile(`(?is)<li[^>]*>(.*?)</li>`)
	s = olRe.ReplaceAllStringFunc(s, func(m string) string {
		inner := olRe.FindStringSubmatch(m)[1]
		counter := 0
		result := liRe.ReplaceAllStringFunc(inner, func(li string) string {
			counter++
			content := strings.TrimSpace(strTags.ReplaceAllString(liRe.FindStringSubmatch(li)[1], ""))
			return fmt.Sprintf("\n%d. %s", counter, content)
		})
		return result + "\n"
	})

	// Convert <ul> list items to bullet markdown.
	ulRe := regexp.MustCompile(`(?is)<ul[^>]*>(.*?)</ul>`)
	s = ulRe.ReplaceAllStringFunc(s, func(m string) string {
		inner := ulRe.FindStringSubmatch(m)[1]
		result := liRe.ReplaceAllStringFunc(inner, func(li string) string {
			content := strings.TrimSpace(strTags.ReplaceAllString(liRe.FindStringSubmatch(li)[1], ""))
			return fmt.Sprintf("\n- %s", content)
		})
		return result + "\n"
	})

	// Inline formatting.
	s = regexp.MustCompile(`(?is)<(em|i)[^>]*>(.*?)</(em|i)>`).ReplaceAllString(s, "*$2*")
	s = regexp.MustCompile(`(?is)<(strong|b)[^>]*>(.*?)</(strong|b)>`).ReplaceAllString(s, "**$2**")
	s = regexp.MustCompile(`(?is)<code[^>]*>(.*?)</code>`).ReplaceAllString(s, "`$1`")

	// Remaining block elements → newline.
	s = regexp.MustCompile(`(?i)</?(?:p|div|br|tr)[^>]*/?>`).ReplaceAllString(s, "\n")

	// Strip any remaining tags.
	s = strTags.ReplaceAllString(s, "")

	// Collapse spaces per line.
	spaceRe := regexp.MustCompile(` {2,}`)
	lines := strings.Split(s, "\n")
	var out []string
	for _, line := range lines {
		out = append(out, strings.TrimSpace(spaceRe.ReplaceAllString(line, " ")))
	}
	s = strings.Join(out, "\n")

	// Collapse 3+ blank lines into 2.
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")

	return strings.TrimSpace(s)
}

// newCreateCmd creates a TestPlan for a story/bug that doesn't have one yet,
// adds a test case derived from the entity content, links bidirectionally,
// and optionally attaches the new plan to a release test plan.
func newCreateCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a test plan for a story or bug, add a test case, and link it bidirectionally",
		ArgsUsage: "<entity-id>",
		UsageText: `# Create and attach to a release test plan
  tp testplan create 363376 --release-plan 365157

  # Create standalone (no release plan)
  tp testplan create 363376`,
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.StringFlag{Name: "name", Usage: "Test plan name (defaults to the entity name)"},
			&cli.IntFlag{Name: "release-plan", Usage: "Release test plan ID to attach this plan to as a child"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) < 1 {
				return errors.New("usage: tp testplan create <entity-id> [--release-plan <id>]")
			}

			entityID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid entity-id %q: %w", args[0], err)
			}

			client, err := f.Client()
			if err != nil {
				return err
			}

			// Auto-detect entity type.
			entityType, err := client.ResolveEntityType(ctx, entityID)
			if err != nil {
				return fmt.Errorf("resolving entity type for %d: %w", entityID, err)
			}

			// Fetch entity including Description.
			entity, err := client.GetEntity(ctx, entityType, entityID, []string{"Name", "Project", "LinkedTestPlan", "Description"})
			if err != nil {
				return fmt.Errorf("fetching entity %d: %w", entityID, err)
			}

			// Guard: already has a test plan.
			if existing, ok := entity["LinkedTestPlan"]; ok && existing != nil {
				if m, ok := existing.(map[string]any); ok && m["Id"] != nil {
					return fmt.Errorf("entity %d already has a linked test plan (ID %v — use 'tp testplan add' to add it as a child)", entityID, m["Id"])
				}
			}

			planName := cmd.String("name")
			if planName == "" {
				if n, ok := entity["Name"].(string); ok {
					planName = n
				} else {
					return errors.New("could not determine entity name; use --name to set the test plan name")
				}
			}

			projectID := 0
			if p, ok := entity["Project"].(map[string]any); ok {
				if id, ok := p["Id"].(float64); ok {
					projectID = int(id)
				}
			}
			if projectID == 0 {
				return errors.New("could not determine project ID from entity")
			}

			// Create the TestPlan linked to the entity.
			plan, err := client.CreateEntity(ctx, "TestPlan", map[string]any{
				"Name":          planName,
				"Project":       map[string]any{"Id": projectID},
				"LinkedGeneral": map[string]any{"Id": entityID},
			})
			if err != nil {
				return fmt.Errorf("creating test plan: %w", err)
			}
			planID := int(plan["Id"].(float64))
			fmt.Printf("Created TestPlan %d %q\n", planID, planName)

			// Link back from the entity to the test plan.
			if _, err = client.UpdateEntity(ctx, entityType, entityID, map[string]any{
				"LinkedTestPlan": map[string]any{"Id": planID},
			}); err != nil {
				return fmt.Errorf("linking test plan back to %s %d: %w", entityType, entityID, err)
			}
			fmt.Printf("Linked to %s %d\n", entityType, entityID)

			// Create a test case from the entity content.
			tcDesc := ""
			if raw, ok := entity["Description"].(string); ok && raw != "" {
				tcDesc = "<!--markdown-->\n" + htmlToMarkdown(raw)
				if len(tcDesc) > 4000 {
					tcDesc = tcDesc[:4000] + "\n…"
				}
			}
			tc, err := client.CreateEntity(ctx, "TestCase", map[string]any{
				"Name":        planName,
				"Description": tcDesc,
				"Project":     map[string]any{"Id": projectID},
			})
			if err != nil {
				return fmt.Errorf("creating test case: %w", err)
			}
			tcID := int(tc["Id"].(float64))

			// Add the test case to the test plan.
			if _, err = client.UpdateEntity(ctx, "TestPlan", planID, map[string]any{
				"TestCases": []map[string]any{{"Id": tcID}},
			}); err != nil {
				return fmt.Errorf("adding test case %d to test plan: %w", tcID, err)
			}
			fmt.Printf("Created TestCase %d and added to plan\n", tcID)

			// Optionally attach to a release test plan.
			if releasePlanID := cmd.Int("release-plan"); releasePlanID > 0 {
				if _, err = client.UpdateEntity(ctx, "TestPlan", releasePlanID, map[string]any{
					"ChildTestPlans": []map[string]any{{"Id": planID}},
				}); err != nil {
					return fmt.Errorf("attaching to release test plan %d: %w", releasePlanID, err)
				}
				fmt.Printf("Attached as child of release TestPlan %d\n", releasePlanID)
			}

			if cmdutil.IsJSON(cmd) {
				return output.PrintJSON(os.Stdout, plan)
			}
			return nil
		},
	}
}

// newAddChildCmd adds one or more child test plans to a parent test plan.
// Uses the ChildTestPlans collection via POST /api/v1/TestPlans/{id}.
func newAddChildCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "Add child test plan(s) to a parent test plan",
		ArgsUsage: "<parent-id> <child-id> [child-id...]",
		UsageText: `# Add one child
  tp testplan add 365157 347067

  # Add multiple children at once
  tp testplan add 365157 347067 360449 361777 360164`,
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) < 2 {
				return errors.New("usage: tp testplan add <parent-id> <child-id> [child-id...]")
			}

			parentID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid parent-id %q: %w", args[0], err)
			}

			childRefs := make([]map[string]any, 0, len(args)-1)
			for _, raw := range args[1:] {
				childID, err := strconv.Atoi(raw)
				if err != nil {
					return fmt.Errorf("invalid child-id %q: %w", raw, err)
				}
				childRefs = append(childRefs, map[string]any{"Id": childID})
			}

			client, err := f.Client()
			if err != nil {
				return err
			}

			result, err := client.UpdateEntity(ctx, "TestPlan", parentID, map[string]any{
				"ChildTestPlans": childRefs,
			})
			if err != nil {
				return fmt.Errorf("adding child test plans: %w", err)
			}

			if cmdutil.IsJSON(cmd) {
				return output.PrintJSON(os.Stdout, result)
			}

			fmt.Printf("Added %d child test plan(s) to TestPlan %d\n", len(childRefs), parentID)
			return nil
		},
	}
}

// newListChildrenCmd lists child test plans of a parent test plan.
func newListChildrenCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:      "list-children",
		Usage:     "List child test plans of a test plan",
		ArgsUsage: "<parent-id>",
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) < 1 {
				return errors.New("usage: tp testplan list-children <parent-id>")
			}

			parentID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid parent-id %q: %w", args[0], err)
			}

			client, err := f.Client()
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/api/v1/TestPlans/%d/ChildTestPlans?format=json", parentID)
			data, err := client.Raw(ctx, "GET", path, nil)
			if err != nil {
				return fmt.Errorf("listing child test plans: %w", err)
			}

			if cmdutil.IsJSON(cmd) {
				_, err = os.Stdout.Write(data)
				return err
			}

			var resp struct {
				Items []struct {
					ID   int    `json:"Id"`
					Name string `json:"Name"`
				} `json:"Items"`
			}
			if err := parseJSON(data, &resp); err != nil {
				_, werr := os.Stdout.Write(data)
				return werr
			}

			if len(resp.Items) == 0 {
				fmt.Printf("No child test plans found for TestPlan %d\n", parentID)
				return nil
			}

			fmt.Printf("%-10s  %s\n", "ID", "Name")
			fmt.Printf("%-10s  %s\n", "----------", "----")
			for _, item := range resp.Items {
				fmt.Printf("%-10d  %s\n", item.ID, item.Name)
			}
			return nil
		},
	}
}

func parseJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
