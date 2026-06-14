package ts

import (
	"sort"

	"github.com/typesense/typesense-go/v4/typesense/api"
	"github.com/verdverm/gmd/pkg/config"
)

// SchemaFieldDiff represents the status of a field compared between config and Typesense.
type SchemaFieldDiff struct {
	Name       string
	ConfigType string // empty for orphaned fields
	TSType     string // empty for pending fields
	Status     string // "OK", "PENDING", "TYPE_MISMATCH", "ORPHANED"
	Facet      bool
	Sort       bool
}

// BaseFields returns the base field names and their types that are always present
// in the Typesense chunks collection schema.
func BaseFields() map[string]string {
	return map[string]string{
		"collection":   "string",
		"path":         "string",
		"title":        "string",
		"content":      "string",
		"hash":         "string",
		"chunk_seq":    "int32",
		"total_chunks": "int32",
		"embedding":    "float[]",
		"links":        "string[]",
	}
}

// DiffSchemaFields compares configured frontmatter fields against actual Typesense fields.
// Returns one entry per configured field (OK, PENDING, TYPE_MISMATCH) followed by
// orphaned fields (in Typesense but not configured and not a base field).
func DiffSchemaFields(configFields map[string]config.FrontmatterField, tsFields []api.Field) []SchemaFieldDiff {
	tsFieldSet := make(map[string]string)
	for _, f := range tsFields {
		tsFieldSet[f.Name] = f.Type
	}

	baseFields := BaseFields()

	// Pre-allocate; at minimum cfgNames entries, plus potential orphans
	diffs := make([]SchemaFieldDiff, 0, len(configFields)+len(tsFields))

	cfgNames := make([]string, 0, len(configFields))
	for name := range configFields {
		cfgNames = append(cfgNames, name)
	}
	sort.Strings(cfgNames)

	for _, fname := range cfgNames {
		f := configFields[fname]
		tsType, inTS := tsFieldSet[fname]
		status := "PENDING"
		if inTS && tsType == f.Type {
			status = "OK"
		} else if inTS && tsType != f.Type {
			status = "TYPE_MISMATCH"
		}
		diffs = append(diffs, SchemaFieldDiff{
			Name:       fname,
			ConfigType: f.Type,
			TSType:     tsType,
			Status:     status,
			Facet:      f.Facet,
			Sort:       f.Sort,
		})
	}

	for _, f := range tsFields {
		if _, isBase := baseFields[f.Name]; isBase {
			continue
		}
		if _, configured := configFields[f.Name]; !configured {
			diffs = append(diffs, SchemaFieldDiff{
				Name:   f.Name,
				TSType: f.Type,
				Status: "ORPHANED",
			})
		}
	}

	return diffs
}
