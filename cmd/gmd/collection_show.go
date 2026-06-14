package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/ts"
)

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
				diffs := ts.DiffSchemaFields(col.Fields, tsFields)
				if len(diffs) > 0 {
					fmt.Println("schema:")
					for _, d := range diffs {
						facetStr := ""
						if d.Facet {
							facetStr = " [facet]"
						}
						sortStr := ""
						if d.Sort {
							sortStr = " [sort]"
						}
						status := d.Status
						if status == "TYPE_MISMATCH" {
							status = fmt.Sprintf("TYPEMISMATCH (TS: %s)", d.TSType)
						}
						fmt.Printf("  %-20s %-8s%s%s  [%s]\n", d.Name, d.ConfigType, facetStr, sortStr, status)
					}
				}
			}
		} else {
			fmt.Printf("type:              wiki\n")
			fmt.Printf("path:              %s\n", wc.Path)
			fmt.Printf("wikiDir:           %s\n", wc.WikiDir)
			fmt.Printf("rawDir:            %s\n", wc.RawDir)
			fmt.Printf("indexFile:         %s\n", wc.IndexFile)
			fmt.Printf("logFile:           %s\n", wc.LogFile)
			fmt.Printf("okfVersion:        %s\n", wc.OkfVersion)
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

			if wc.Frontmatter != nil {
				fmt.Println("frontmatter (wiki-only):")
				if wc.Frontmatter.Type != "" {
					fmt.Printf("  type:              %s (required)\n", wc.Frontmatter.Type)
				}
				if wc.Frontmatter.Title != "" {
					fmt.Printf("  title:             %s\n", wc.Frontmatter.Title)
				}
				if wc.Frontmatter.Description != "" {
					fmt.Printf("  description:       %s\n", wc.Frontmatter.Description)
				}
				if wc.Frontmatter.Resource != "" {
					fmt.Printf("  resource:          %s\n", wc.Frontmatter.Resource)
				}
				if len(wc.Frontmatter.Tags) > 0 {
					fmt.Printf("  tags:              %v\n", wc.Frontmatter.Tags)
				}
				if wc.Frontmatter.Timestamp != "" {
					fmt.Printf("  timestamp:         %s\n", wc.Frontmatter.Timestamp)
				}
				if wc.Frontmatter.Status != "" {
					fmt.Printf("  status:            %s\n", wc.Frontmatter.Status)
				}
				if len(wc.Frontmatter.Sources) > 0 {
					fmt.Printf("  sources:           %v\n", wc.Frontmatter.Sources)
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
