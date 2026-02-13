package entity

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/api"
	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
	"github.com/lifedraft/targetprocess-cli/internal/text"
)

func newCommentCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "comment",
		Usage: "Manage comments on entities",
		UsageText: `# List comments on an entity
  tp entity comment list --entity-id 342236

  # Add a comment with @mentions
  tp entity comment add --entity-id 342236 --body "Hey @timo, this looks good"

  # Delete a comment
  tp entity comment delete --id 99999`,
		Commands: []*cli.Command{
			newCommentListCmd(f),
			newCommentAddCmd(f),
			newCommentDeleteCmd(f),
		},
	}
}

func newCommentListCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List comments on an entity",
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.IntFlag{Name: "entity-id", Required: true, Usage: "Entity ID to list comments for"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			entityID := cmd.Int("entity-id")
			if entityID <= 0 {
				return fmt.Errorf("entity ID must be positive, got %d", entityID)
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

func newCommentAddCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Add a comment to an entity",
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.IntFlag{Name: "entity-id", Required: true, Usage: "Entity ID to comment on"},
			&cli.StringFlag{Name: "body", Required: true, Usage: "Comment text"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			entityID := cmd.Int("entity-id")
			if entityID <= 0 {
				return fmt.Errorf("entity ID must be positive, got %d", entityID)
			}

			body := cmd.String("body")
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

func newCommentDeleteCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a comment by ID",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "id", Required: true, Usage: "Comment ID to delete"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			id := cmd.Int("id")
			if id <= 0 {
				return fmt.Errorf("comment ID must be positive, got %d", id)
			}

			if _, err := client.DeleteEntity(ctx, "Comment", id); err != nil {
				return fmt.Errorf("deleting comment: %w", err)
			}

			fmt.Fprintf(os.Stdout, "Deleted comment %d\n", id)
			return nil
		},
	}
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
		// Strip <!--markdown--> prefix for display
		desc = strings.TrimPrefix(desc, "<!--markdown-->")
		desc = strings.TrimSpace(desc)
		// Truncate to ~80 chars
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}

		fmt.Fprintf(tw, "%v\t%s\t%s\t%s\n", id, owner, date, desc)
	}
	tw.Flush()
}
