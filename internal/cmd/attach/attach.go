package attachcmd

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

// NewCmd creates the "attach" command, which uploads file attachments to an
// entity via Targetprocess's UploadFile.ashx endpoint.
func NewCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:      "attach",
		Usage:     "Upload file attachments to an entity",
		ArgsUsage: "<entity-id> <file>...",
		UsageText: `# Attach a screenshot to a test case run
  tp attach 57755 ./evidence.png

  # Attach multiple files with a message
  tp attach 363529 a.png b.png --message "Release test evidence"`,
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.StringFlag{Name: "message", Aliases: []string{"m"}, Usage: "Optional message stored with the attachment"},
			&cli.IntFlag{Name: "entity-id", Usage: "Entity ID (alternative to positional argument)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			entityID, files, err := resolveArgs(cmd)
			if err != nil {
				return err
			}

			// Validate files up front so we fail before touching the API.
			for _, fp := range files {
				info, statErr := os.Stat(fp)
				if statErr != nil {
					return fmt.Errorf("cannot read file %q: %w", fp, statErr)
				}
				if info.IsDir() {
					return fmt.Errorf("%q is a directory, not a file", fp)
				}
			}

			client, err := f.Client()
			if err != nil {
				return err
			}

			data, err := client.UploadAttachment(ctx, entityID, files, cmd.String("message"))
			if err != nil {
				return fmt.Errorf("uploading attachment: %w", err)
			}

			if cmdutil.IsJSON(cmd) {
				var parsed any
				if json.Valid(data) {
					if uerr := json.Unmarshal(data, &parsed); uerr == nil {
						return output.PrintJSON(os.Stdout, parsed)
					}
				}
				fmt.Fprintln(os.Stdout, string(data))
				return nil
			}

			printResult(data, entityID)
			return nil
		},
	}
}

func resolveArgs(cmd *cli.Command) (entityID int, files []string, err error) {
	args := cmd.Args().Slice()

	if flagID := cmd.Int("entity-id"); flagID > 0 {
		if len(args) == 0 {
			return 0, nil, errors.New("at least one file is required; usage: tp attach --entity-id <id> <file>...")
		}
		return flagID, args, nil
	}

	if len(args) < 2 {
		return 0, nil, errors.New("entity ID and at least one file are required; usage: tp attach <entity-id> <file>...")
	}

	entityID, err = strconv.Atoi(args[0])
	if err != nil {
		return 0, nil, fmt.Errorf("invalid entity ID %q: must be an integer", args[0])
	}
	if entityID <= 0 {
		return 0, nil, fmt.Errorf("entity ID must be positive, got %d", entityID)
	}
	return entityID, args[1:], nil
}

func printResult(data []byte, entityID int) {
	var resp struct {
		Items []struct {
			ID            int    `json:"id"`
			Name          string `json:"name"`
			Size          int64  `json:"size"`
			PersistedSize int64  `json:"persistedSize"`
			URI           string `json:"uri"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || len(resp.Items) == 0 {
		// Unexpected shape: print the raw response so nothing is hidden.
		fmt.Fprintln(os.Stdout, string(data))
		return
	}

	tw := output.NewTabWriter(os.Stdout)
	fmt.Fprintln(tw, "ID\tNAME\tSIZE\tURI")
	for _, it := range resp.Items {
		size := it.Size
		if size == 0 {
			size = it.PersistedSize
		}
		fmt.Fprintf(tw, "%d\t%s\t%d\t%s\n", it.ID, it.Name, size, it.URI)
	}
	tw.Flush()
	fmt.Fprintf(os.Stdout, "\nUploaded %d file(s) to entity %d\n", len(resp.Items), entityID)
}
