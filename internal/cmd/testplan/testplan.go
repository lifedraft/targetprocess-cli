package testplan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
)

// NewCmd creates the "testplan" command.
func NewCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "testplan",
		Usage: "Manage test plans",
		UsageText: `# Add child test plans to a parent test plan
  tp testplan add 365157 347067 360449 361777

  # List child test plans of a test plan
  tp testplan list-children 365157`,
		Commands: []*cli.Command{
			newAddChildCmd(f),
			newListChildrenCmd(f),
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

			// Parse and print table
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
