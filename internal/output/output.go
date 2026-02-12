package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
)

// PrintJSON writes v as pretty-printed JSON to w.
func PrintJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintEntity prints a single entity as key-value pairs.
func PrintEntity(w io.Writer, entity map[string]any) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, key := range sortedKeys(entity) {
		val := entity[key]
		switch v := val.(type) {
		case map[string]any:
			if name, ok := v["Name"]; ok {
				fmt.Fprintf(tw, "%s:\t%v\n", key, name)
			} else if id, ok := v["Id"]; ok {
				fmt.Fprintf(tw, "%s:\t%v\n", key, id)
			} else {
				fmt.Fprintf(tw, "%s:\t%v\n", key, v)
			}
		default:
			fmt.Fprintf(tw, "%s:\t%v\n", key, val)
		}
	}
	tw.Flush()
}

// PrintEntityTable prints a list of entities as a table.
func PrintEntityTable(w io.Writer, entities []map[string]any) {
	if len(entities) == 0 {
		fmt.Fprintln(w, "No results found.")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "ID\tNAME\tTYPE\tSTATE\n")
	for _, e := range entities {
		id := e["Id"]
		name := e["Name"]
		rtype := e["ResourceType"]
		state := ""
		if es, ok := e["EntityState"].(map[string]any); ok {
			if n, ok := es["Name"]; ok {
				state = fmt.Sprintf("%v", n)
			}
		}
		fmt.Fprintf(tw, "%v\t%v\t%v\t%v\n", id, name, rtype, state)
	}
	tw.Flush()
}

// PrintMetaTypes prints entity type metadata as a table.
func PrintMetaTypes(w io.Writer, types []string) {
	for _, t := range types {
		fmt.Fprintln(w, t)
	}
}

// PrintProperties prints entity properties as a table.
func PrintProperties(w io.Writer, props []map[string]string) {
	if len(props) == 0 {
		fmt.Fprintln(w, "No properties found.")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "NAME\tTYPE\tNULLABLE\n")
	for _, p := range props {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", p["name"], p["type"], p["nullable"])
	}
	tw.Flush()
}

// NewTabWriter creates a new tabwriter with standard formatting settings.
func NewTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
