package api //nolint:revive // package name matches directory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
)

func NewCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:      "api",
		Usage:     "Make raw API requests to Targetprocess",
		ArgsUsage: "<method> <path>",
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.StringFlag{Name: "body", Usage: "Request body (JSON string)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			args := cmd.Args().Slice()
			if len(args) == 0 {
				return errors.New("path is required; usage: tp api [METHOD] <path>")
			}

			var method, path string
			if len(args) == 1 {
				method = "GET"
				path = args[0]
			} else {
				method = strings.ToUpper(args[0])
				path = args[1]
			}

			bodyStr := cmd.String("body")
			var bodyReader *strings.Reader
			if bodyStr != "" {
				bodyReader = strings.NewReader(bodyStr)
			}

			var data []byte
			if bodyReader != nil {
				data, err = client.Raw(ctx, method, path, bodyReader)
			} else {
				data, err = client.Raw(ctx, method, path, nil)
			}
			if err != nil {
				return fmt.Errorf("API request failed: %w", err)
			}

			var parsed any
			if err := json.Unmarshal(data, &parsed); err != nil {
				// Not valid JSON, print raw
				fmt.Fprintln(os.Stdout, string(data))
				return err
			}

			return output.PrintJSON(os.Stdout, parsed)
		},
	}
}
