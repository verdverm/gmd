package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

type tsFieldInfo struct {
	Name string
	Type string
}

func printSchemaDiff(fields map[string]config.FrontmatterField, tsFields []tsFieldInfo) {
	tsFieldSet := make(map[string]string)
	for _, f := range tsFields {
		tsFieldSet[f.Name] = f.Type
	}

	baseFields := map[string]string{
		"collection": "string", "path": "string", "title": "string",
		"content": "string", "hash": "string", "chunk_seq": "int32",
		"total_chunks": "int32", "embedding": "float[]", "links": "string[]",
	}

	if len(fields) > 0 || len(tsFields) > len(baseFields) {
		fmt.Println("schema:")
		for _, fname := range sortedKeys(fields) {
			f := fields[fname]
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
		for _, f := range tsFields {
			if _, isBase := baseFields[f.Name]; isBase {
				continue
			}
			if _, configured := fields[f.Name]; !configured {
				fmt.Printf("  %-20s %-8s           [ORPHANED]\n", f.Name, f.Type)
			}
		}
	}
}

var collectionShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show collection or wiki config details, chunk count, and schema",
	Long: `Displays the full configuration for a collection or wiki including path, pattern,
ignore rules, excludeFromDefault, and frontmatter fields — along
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
		cfg := r.Config()

		col, isCol := cfg.Collections[name]
		wc, isWiki := cfg.Wikis[name]
		if !isCol && !isWiki {
			return fmt.Errorf("source %q not found (not a collection or wiki)", name)
		}

		ctx := context.Background()

		fmt.Printf("name:              %s\n", name)

		if isCol {
			fmt.Printf("type:              collection\n")
			fmt.Printf("path:              %s\n", col.Path)
			fmt.Printf("patterns:          %v\n", col.Patterns)
			if len(col.Ignore) > 0 {
				fmt.Printf("ignore:            %v\n", col.Ignore)
			}
			fmt.Printf("excludeFromDefault: %v\n", col.ExcludeFromDefault)

			if len(col.Fields) > 0 {
				fmt.Println("fields (config):")
				for _, fname := range sortedKeys(col.Fields) {
					f := col.Fields[fname]
					facetStr := ""
					if f.Facet {
						facetStr = " [facet]"
					}
					sortStr := ""
					if f.Sort {
						sortStr = " [sort]"
					}
					fmt.Printf("  %-20s %-8s%s%s\n", fname, f.Type, facetStr, sortStr)
				}
			}

			tsFields, err := r.TSClient().GetSchemaFields(ctx)
			if err != nil {
				fmt.Printf("schema: (error fetching Typesense schema: %v)\n", err)
			} else {
				tf := make([]tsFieldInfo, len(tsFields))
				for i, f := range tsFields {
					tf[i] = tsFieldInfo{Name: f.Name, Type: f.Type}
				}
				printSchemaDiff(col.Fields, tf)
			}
		} else {
			fmt.Printf("type:              wiki\n")
			fmt.Printf("path:              %s\n", wc.Path)
			fmt.Printf("wikiDir:           %s\n", wc.WikiDir)
			fmt.Printf("rawDir:            %s\n", wc.RawDir)
			fmt.Printf("indexFile:         %s\n", wc.IndexFile)
			fmt.Printf("logFile:           %s\n", wc.LogFile)
			fmt.Printf("graphLinks:        %v\n", wc.GraphLinks)
			fmt.Printf("patterns:          %v\n", wc.Patterns)
			if len(wc.Ignore) > 0 {
				fmt.Printf("ignore:            %v\n", wc.Ignore)
			}
			fmt.Printf("excludeFromDefault: %v\n", wc.ExcludeFromDefault)
			if len(wc.SourceRefs) > 0 {
				fmt.Printf("sourceRefs:        %v\n", wc.SourceRefs)
			}

			if len(wc.Fields) > 0 {
				fmt.Println("fields (config):")
				for _, fname := range sortedKeys(wc.Fields) {
					f := wc.Fields[fname]
					facetStr := ""
					if f.Facet {
						facetStr = " [facet]"
					}
					sortStr := ""
					if f.Sort {
						sortStr = " [sort]"
					}
					fmt.Printf("  %-20s %-8s%s%s\n", fname, f.Type, facetStr, sortStr)
				}
			}

			if wc.Frontmatter != nil && len(wc.Frontmatter.Fields) > 0 {
				fmt.Println("frontmatter (wiki-only):")
				for _, fname := range sortedKeys(wc.Frontmatter.Fields) {
					f := wc.Frontmatter.Fields[fname]
					fmt.Printf("  %-20s %-8s\n", fname, f.Type)
				}
			}
		}

		key := cfg.CollectionKey(name)
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
