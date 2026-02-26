package commentcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/api"
	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
	"github.com/lifedraft/targetprocess-cli/internal/text"
)

// NewCmd creates the "comment" command.
func NewCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "comment",
		Usage: "Manage comments on entities",
		UsageText: `# List comments on an entity
  tp comment list 342236

  # Add a comment with @mentions
  tp comment add 342236 "Hey @timo, this looks good"

  # Delete a comment
  tp comment delete 99999`,
		Commands: []*cli.Command{
			newListCmd(f),
			newAddCmd(f),
			newDeleteCmd(f),
		},
	}
}

func newListCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "List comments on an entity",
		ArgsUsage: "<entity-id>",
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.IntFlag{Name: "entity-id", Usage: "Entity ID (alternative to positional argument)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			entityID, err := resolveEntityID(cmd)
			if err != nil {
				return err
			}

			client, err := f.Client()
			if err != nil {
				return err
			}

			where := fmt.Sprintf("General.Id eq %d", entityID)
			include := []string{"Description", "CreateDate", "Owner"}

			comments, err := client.SearchEntities(ctx, "Comment", where, include, 0, nil)
			if err != nil {
				return fmt.Errorf("listing comments: %w", err)
			}

			if cmdutil.IsJSON(cmd) {
				return output.PrintJSON(os.Stdout, map[string]any{
					"items": comments,
					"count": len(comments),
				})
			}

			printCommentTable(comments)
			return nil
		},
	}
}

func newAddCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "Add a comment to an entity",
		ArgsUsage: "<entity-id> <body>",
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.IntFlag{Name: "entity-id", Usage: "Entity ID (alternative to positional argument)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()

			entityID, body, err := resolveAddArgs(cmd, args)
			if err != nil {
				return err
			}

			client, err := f.Client()
			if err != nil {
				return err
			}

			fields := map[string]any{
				"Description": body,
				"General":     map[string]any{"Id": entityID},
			}

			if prepErr := text.PrepareFields(ctx, client, fields); prepErr != nil {
				return fmt.Errorf("preparing comment fields: %w", prepErr)
			}

			entity, err := client.CreateEntity(ctx, "Comment", fields)
			if err != nil {
				return fmt.Errorf("adding comment: %w", err)
			}

			if cmdutil.IsJSON(cmd) {
				return output.PrintJSON(os.Stdout, entity)
			}

			output.PrintEntity(os.Stdout, entity)
			return nil
		},
	}
}

func newDeleteCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a comment by ID",
		ArgsUsage: "<comment-id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) == 0 {
				return errors.New("comment ID is required; usage: tp comment delete <comment-id>")
			}

			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid comment ID %q: must be an integer", args[0])
			}
			if id <= 0 {
				return fmt.Errorf("comment ID must be positive, got %d", id)
			}

			client, err := f.Client()
			if err != nil {
				return err
			}

			if _, err := client.DeleteEntity(ctx, "Comment", id); err != nil {
				return fmt.Errorf("deleting comment: %w", err)
			}

			fmt.Fprintf(os.Stdout, "Deleted comment %d\n", id)
			return nil
		},
	}
}

func resolveEntityID(cmd *cli.Command) (int, error) {
	args := cmd.Args().Slice()
	if len(args) > 0 {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return 0, fmt.Errorf("invalid entity ID %q: must be an integer", args[0])
		}
		if id <= 0 {
			return 0, fmt.Errorf("entity ID must be positive, got %d", id)
		}
		return id, nil
	}

	if id := cmd.Int("entity-id"); id > 0 {
		return id, nil
	}

	return 0, errors.New("entity ID is required; usage: tp comment list <entity-id> or tp comment list --entity-id <id>")
}

func resolveAddArgs(cmd *cli.Command, args []string) (entityID int, body string, err error) {
	if len(args) >= 2 {
		entityID, err = strconv.Atoi(args[0])
		if err != nil {
			return 0, "", fmt.Errorf("invalid entity ID %q: must be an integer", args[0])
		}
		if entityID <= 0 {
			return 0, "", fmt.Errorf("entity ID must be positive, got %d", entityID)
		}
		return entityID, args[1], nil
	}

	// Try --entity-id flag + single positional arg as body
	if flagID := cmd.Int("entity-id"); flagID > 0 {
		entityID = flagID
		if len(args) >= 1 {
			return entityID, args[0], nil
		}
		return 0, "", errors.New("comment body is required; usage: tp comment add --entity-id <id> <body>")
	}

	return 0, "", errors.New("entity ID and comment body are required; usage: tp comment add <entity-id> <body>")
}

func printCommentTable(comments []api.Entity) {
	if len(comments) == 0 {
		fmt.Fprintln(os.Stdout, "No comments found.")
		return
	}

	tw := output.NewTabWriter(os.Stdout)
	fmt.Fprintln(tw, "ID\tOWNER\tDATE\tDESCRIPTION")

	for _, c := range comments {
		id := c["Id"]
		owner := ""
		if o, ok := c["Owner"].(map[string]any); ok {
			if name, ok := o["Name"]; ok {
				owner = fmt.Sprintf("%v", name)
			}
		}
		date := ""
		if d, ok := c["CreateDate"]; ok {
			date = fmt.Sprintf("%v", d)
		}
		desc := ""
		if d, ok := c["Description"]; ok {
			desc = fmt.Sprintf("%v", d)
		}
		desc = strings.TrimPrefix(desc, "<!--markdown-->")
		desc = strings.TrimSpace(desc)
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}

		fmt.Fprintf(tw, "%v\t%s\t%s\t%s\n", id, owner, date, desc)
	}
	tw.Flush()
}
