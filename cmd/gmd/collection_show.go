package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var collectionShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show collection config details, chunk count, and schema",
	Long: `Displays the full configuration for a collection including path, pattern,
ignore rules, context, includeByDefault, and frontmatter fields — along
with the current chunk count and actual Typesense schema fields.

Example:
  gmd collection show mydocs`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}

		name := args[0]
		col, ok := r.Config().Collections[name]
		if !ok {
			return fmt.Errorf("collection %q not found", name)
		}

		ctx := context.Background()

		fmt.Printf("name:              %s\n", name)
		fmt.Printf("path:              %s\n", col.Path)
		fmt.Printf("patterns:          %v\n", col.Patterns)
		if len(col.Ignore) > 0 {
			fmt.Printf("ignore:            %v\n", col.Ignore)
		}
		if col.Context != "" {
			fmt.Printf("context:           %s\n", col.Context)
		}
		fmt.Printf("includeByDefault:  %v\n", col.IncludeByDefault)

		// Print configured frontmatter fields
		if len(col.Fields) > 0 {
			fmt.Println("fields (config):")
			for _, name := range sortedKeys(col.Fields) {
				f := col.Fields[name]
				facetStr := ""
				if f.Facet {
					facetStr = " [facet]"
				}
				sortStr := ""
				if f.Sort {
					sortStr = " [sort]"
				}
				fmt.Printf("  %-20s %-8s%s%s\n", name, f.Type, facetStr, sortStr)
			}
		}

		// Diff configured fields against actual Typesense schema
		tsFields, err := r.TSClient().GetSchemaFields(ctx)
		if err != nil {
			fmt.Printf("schema: (error fetching Typesense schema: %v)\n", err)
		} else {
			tsFieldSet := make(map[string]string)
			for _, f := range tsFields {
				tsFieldSet[f.Name] = f.Type
			}

			baseFields := map[string]string{
				"collection": "string", "path": "string", "title": "string",
				"content": "string", "hash": "string", "chunk_seq": "int32",
				"total_chunks": "int32", "embedding": "float[]", "links": "string[]",
			}

			if len(col.Fields) > 0 || len(tsFields) > len(baseFields) {
				fmt.Println("schema:")
				// Show configured fields with sync status
				for _, fname := range sortedKeys(col.Fields) {
					f := col.Fields[fname]
					tsType, inTS := tsFieldSet[fname]
					status := "PENDING"
					if inTS && tsType == f.Type {
						status = "OK"
					} else if inTS && tsType != f.Type {
						status = fmt.Sprintf("TYPEMISMATCH (TS: %s)", tsType)
					}
					facetStr := ""
					if f.Facet {
						facetStr = " [facet]"
					}
					sortStr := ""
					if f.Sort {
						sortStr = " [sort]"
					}
					fmt.Printf("  %-20s %-8s%s%s  [%s]\n", fname, f.Type, facetStr, sortStr, status)
				}
				// Show orphaned TS fields (non-base, not in config)
				for _, f := range tsFields {
					if _, isBase := baseFields[f.Name]; isBase {
						continue
					}
					if _, configured := col.Fields[f.Name]; !configured {
						fmt.Printf("  %-20s %-8s           [ORPHANED]\n", f.Name, f.Type)
					}
				}
			}
		}

		key := r.Config().CollectionKey(name)
		counts, err := r.TSClient().CountByCollection(ctx, []string{key})
		if err != nil {
			fmt.Printf("chunks:            (error counting: %v)\n", err)
		} else {
			fmt.Printf("chunks:            %d\n", counts[key])
		}
		return nil
	},
}

func init() {
	collectionCmd.AddCommand(collectionShowCmd)
}
