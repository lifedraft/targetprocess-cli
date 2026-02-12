package inspect

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/lifedraft/targetprocess-cli/internal/api"
	"github.com/lifedraft/targetprocess-cli/internal/cmdutil"
	"github.com/lifedraft/targetprocess-cli/internal/output"
)

// XML structures for TP metadata

type metaIndex struct {
	XMLName xml.Name        `xml:"ResourceMetadataDescriptionIndex"`
	Types   []metaIndexItem `xml:"ResourceMetadataDescription"`
}

type metaIndexItem struct {
	Name        string `xml:"Name,attr"`
	URI         string `xml:"Uri,attr"` //nolint:revive // XML tag must match API response
	Description string `xml:"Description,attr"`
}

type typeMeta struct {
	XMLName    xml.Name       `xml:"ResourceMetadataDescription"`
	Name       string         `xml:"Name,attr"`
	Properties typeProperties `xml:"ResourceMetadataPropertiesDescription"`
}

type typeProperties struct {
	Values      []fieldMeta `xml:"ResourceMetadataPropertiesResourceValuesDescription>ResourceFieldMetadataDescription"`
	References  []fieldMeta `xml:"ResourceMetadataPropertiesResourceReferencesDescription>ResourceFieldMetadataDescription"`
	Collections []fieldMeta `xml:"ResourceMetadataPropertiesResourceCollectionsDescription>ResourceCollecitonFieldMetadataDescription"`
}

type fieldMeta struct {
	Name        string `xml:"Name,attr"`
	Type        string `xml:"Type,attr"`
	CanSet      string `xml:"CanSet,attr"`
	CanGet      string `xml:"CanGet,attr"`
	IsRequired  string `xml:"IsRequired,attr"`
	Description string `xml:"Description,attr"`
}

func NewCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "inspect",
		Usage: "Inspect Targetprocess API metadata (entity types, properties)",
		Commands: []*cli.Command{
			newTypesCmd(f),
			newPropertiesCmd(f),
			newDetailsCmd(f),
			newDiscoverCmd(f),
		},
	}
}

func newTypesCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "types",
		Usage: "List all available entity types",
		Flags: []cli.Flag{cmdutil.OutputFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			data, err := client.GetMetaIndex(ctx)
			if err != nil {
				return fmt.Errorf("fetching metadata: %w", err)
			}

			var index metaIndex
			if err := xml.Unmarshal(data, &index); err != nil {
				return fmt.Errorf("parsing metadata XML: %w", err)
			}

			if cmdutil.IsJSON(cmd) {
				types := make([]map[string]string, len(index.Types))
				for i, t := range index.Types {
					types[i] = map[string]string{
						"name":        t.Name,
						"description": t.Description,
					}
				}
				return output.PrintJSON(os.Stdout, map[string]any{"types": types})
			}

			names := make([]string, len(index.Types))
			for i, t := range index.Types {
				names[i] = t.Name
			}
			sort.Strings(names)
			output.PrintMetaTypes(os.Stdout, names)
			return nil
		},
	}
}

func newPropertiesCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "properties",
		Usage: "List properties of an entity type",
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.StringFlag{Name: "type", Required: true, Usage: "Entity type (e.g. UserStory)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			entityType := cmd.String("type")
			data, err := client.GetTypeMeta(ctx, entityType)
			if err != nil {
				return fmt.Errorf("fetching type metadata: %w", err)
			}

			var meta typeMeta
			if err := xml.Unmarshal(data, &meta); err != nil {
				return fmt.Errorf("parsing type metadata XML: %w", err)
			}

			allFields := append(append(meta.Properties.Values, meta.Properties.References...), meta.Properties.Collections...)

			if cmdutil.IsJSON(cmd) {
				fields := make([]map[string]string, len(allFields))
				for i, f := range allFields {
					fields[i] = map[string]string{
						"name":        f.Name,
						"type":        f.Type,
						"canSet":      f.CanSet,
						"canGet":      f.CanGet,
						"isRequired":  f.IsRequired,
						"description": f.Description,
					}
				}
				return output.PrintJSON(os.Stdout, map[string]any{"properties": fields})
			}

			props := make([]map[string]string, len(allFields))
			for i, f := range allFields {
				props[i] = map[string]string{
					"name":     f.Name,
					"type":     f.Type,
					"nullable": strconv.FormatBool(f.IsRequired != "true"),
				}
			}
			output.PrintProperties(os.Stdout, props)
			return nil
		},
	}
}

func newDiscoverCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "discover",
		Usage: "Discover API structure and available entity types",
		Flags: []cli.Flag{cmdutil.OutputFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			// Try primary method: fetch metadata index
			types, source, err := discoverFromMeta(ctx, client)
			if err != nil {
				// Fallback: trigger informative error
				types, source, err = discoverFromError(ctx, client)
				if err != nil {
					return fmt.Errorf("failed to discover API structure: %w", err)
				}
			}

			if cmdutil.IsJSON(cmd) {
				return output.PrintJSON(os.Stdout, map[string]any{
					"entityTypes": types,
					"source":      source,
				})
			}

			fmt.Fprintf(os.Stdout, "Discovered %d entity types (source: %s)\n\n", len(types), source)
			for _, t := range types {
				fmt.Fprintln(os.Stdout, t)
			}
			return nil
		},
	}
}

func discoverFromMeta(ctx context.Context, client *api.Client) (names []string, source string, err error) {
	data, err := client.GetMetaIndex(ctx)
	if err != nil {
		return nil, "", err
	}

	var index metaIndex
	if err := xml.Unmarshal(data, &index); err != nil {
		return nil, "", fmt.Errorf("parsing metadata XML: %w", err)
	}

	names = make([]string, len(index.Types))
	for i, t := range index.Types {
		names[i] = t.Name
	}
	sort.Strings(names)
	return names, "metadata", nil
}

func discoverFromError(ctx context.Context, client *api.Client) (types []string, source string, err error) {
	_, err = client.GetEntity(ctx, "NonExistentType", 1, nil)
	if err == nil {
		return nil, "", errors.New("unexpected: no error for non-existent type")
	}

	errMsg := err.Error()

	// Look for pattern like "Valid entity types are: Bug, Epic, Feature, ..."
	marker := "Valid entity types are:"
	idx := strings.Index(errMsg, marker)
	if idx < 0 {
		// Try alternate patterns
		marker = "valid entity types are:"
		idx = strings.Index(strings.ToLower(errMsg), strings.ToLower(marker))
	}
	if idx < 0 {
		return nil, "", fmt.Errorf("could not extract entity types from error: %s", errMsg)
	}

	listStr := errMsg[idx+len(marker):]
	// Trim any trailing punctuation or whitespace
	listStr = strings.TrimSpace(listStr)
	listStr = strings.TrimRight(listStr, ".")

	parts := strings.Split(listStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			types = append(types, p)
		}
	}
	sort.Strings(types)
	return types, "error_discovery", nil
}

func newDetailsCmd(f *cmdutil.Factory) *cli.Command {
	return &cli.Command{
		Name:  "details",
		Usage: "Get detailed info about an entity property",
		Flags: []cli.Flag{
			cmdutil.OutputFlag(),
			&cli.StringFlag{Name: "type", Required: true, Usage: "Entity type (e.g. UserStory)"},
			&cli.StringFlag{Name: "property", Required: true, Usage: "Property name (e.g. EntityState)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			entityType := cmd.String("type")
			propName := cmd.String("property")

			data, err := client.GetTypeMeta(ctx, entityType)
			if err != nil {
				return fmt.Errorf("fetching type metadata: %w", err)
			}

			var meta typeMeta
			if err := xml.Unmarshal(data, &meta); err != nil {
				return fmt.Errorf("parsing type metadata XML: %w", err)
			}

			allFields := append(append(meta.Properties.Values, meta.Properties.References...), meta.Properties.Collections...)
			for _, f := range allFields {
				if f.Name == propName {
					detail := map[string]any{
						"Name":        f.Name,
						"Type":        f.Type,
						"CanSet":      f.CanSet,
						"CanGet":      f.CanGet,
						"IsRequired":  f.IsRequired,
						"Description": f.Description,
					}
					if cmdutil.IsJSON(cmd) {
						return output.PrintJSON(os.Stdout, detail)
					}
					output.PrintEntity(os.Stdout, detail)
					return nil
				}
			}
			return fmt.Errorf("property %q not found on type %q", propName, entityType)
		},
	}
}
