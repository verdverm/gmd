package config

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/parser"
)

// ProjectConfigPath returns the path to the project config file.
func ProjectConfigPath(root string) string {
	return filepath.Join(root, sentinelDir, "config.cue")
}

func readConfigFile(path string) (*ast.File, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	f, err := parser.ParseFile(path, src)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return f, nil
}

func writeConfigFile(path string, f *ast.File) error {
	src, err := format.Node(f, format.Simplify())
	if err != nil {
		return fmt.Errorf("formatting config: %w", err)
	}
	return os.WriteFile(path, src, 0644)
}

func fieldLabel(f *ast.Field) string {
	name, _, _ := ast.LabelName(f.Label)
	return name
}

func findFieldInStruct(st *ast.StructLit, label string) *ast.Field {
	for _, d := range st.Elts {
		f, ok := d.(*ast.Field)
		if !ok {
			continue
		}
		if fieldLabel(f) == label {
			return f
		}
	}
	return nil
}

func getOrCreateStruct(st *ast.StructLit, label string) *ast.StructLit {
	f := findFieldInStruct(st, label)
	if f != nil {
		if inner, ok := f.Value.(*ast.StructLit); ok {
			return inner
		}
		inner := &ast.StructLit{}
		f.Value = inner
		return inner
	}
	inner := &ast.StructLit{}
	st.Elts = append(st.Elts, &ast.Field{
		Label: ast.NewIdent(label),
		Value: inner,
	})
	return inner
}

func getConfigStruct(f *ast.File) *ast.StructLit {
	for _, d := range f.Decls {
		if fld, ok := d.(*ast.Field); ok && fieldLabel(fld) == "Config" {
			if st, ok := fld.Value.(*ast.StructLit); ok {
				return st
			}
		}
	}
	return nil
}

func strField(name, value string) *ast.Field {
	return &ast.Field{
		Label: ast.NewIdent(name),
		Value: ast.NewString(value),
	}
}

func strListField(name string, values []string) *ast.Field {
	list := &ast.ListLit{}
	for _, v := range values {
		list.Elts = append(list.Elts, ast.NewString(v))
	}
	return &ast.Field{
		Label: ast.NewIdent(name),
		Value: list,
	}
}

func boolField(name string, value bool) *ast.Field {
	return &ast.Field{
		Label: ast.NewIdent(name),
		Value: ast.NewBool(value),
	}
}

// AddCollection adds a new collection to the project config file.
func AddCollection(cfg *Config, name, path string, patterns []string) error {
	if _, exists := cfg.Collections[name]; exists {
		return fmt.Errorf("collection %q already exists", name)
	}

	cfgPath := ProjectConfigPath(cfg.ProjectRoot)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return fmt.Errorf("no config file found at %s", cfgPath)
	}

	f, err := readConfigFile(cfgPath)
	if err != nil {
		return err
	}

	cs := getConfigStruct(f)
	if cs == nil {
		return fmt.Errorf("no Config block found in %s", cfgPath)
	}

	cols := getOrCreateStruct(cs, "collections")
	if findFieldInStruct(cols, name) != nil {
		return fmt.Errorf("collection %q already exists in config", name)
	}

	colStruct := &ast.StructLit{}
	colStruct.Elts = append(colStruct.Elts,
		strField("path", path),
		strListField("patterns", patterns),
	)

	cols.Elts = append(cols.Elts, &ast.Field{
		Label: ast.NewIdent(name),
		Value: colStruct,
	})

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	// Update in-memory config
	cfg.Collections[name] = CollectionConfig{
		Path:             path,
		Patterns:         patterns,
		IncludeByDefault: true,
	}

	return nil
}

// RemoveCollection removes a collection from the project config file and
// returns the removed collection config.
func RemoveCollection(cfg *Config, name string) error {
	if _, exists := cfg.Collections[name]; !exists {
		return fmt.Errorf("collection %q not found", name)
	}

	cfgPath := ProjectConfigPath(cfg.ProjectRoot)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return fmt.Errorf("no config file found at %s", cfgPath)
	}

	f, err := readConfigFile(cfgPath)
	if err != nil {
		return err
	}

	cs := getConfigStruct(f)
	if cs == nil {
		return fmt.Errorf("no Config block found in %s", cfgPath)
	}

	cols := findFieldInStruct(cs, "collections")
	if cols == nil {
		return fmt.Errorf("no collections found in config")
	}
	colSt, ok := cols.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("collections is not a struct")
	}

	found := false
	for i, d := range colSt.Elts {
		if fld, ok := d.(*ast.Field); ok && fieldLabel(fld) == name {
			colSt.Elts = append(colSt.Elts[:i], colSt.Elts[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("collection %q not found in config", name)
	}

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	delete(cfg.Collections, name)
	return nil
}

// RenameCollection renames a collection in the project config file.
func RenameCollection(cfg *Config, oldName, newName string) error {
	if _, exists := cfg.Collections[oldName]; !exists {
		return fmt.Errorf("collection %q not found", oldName)
	}
	if _, exists := cfg.Collections[newName]; exists {
		return fmt.Errorf("collection %q already exists", newName)
	}

	cfgPath := ProjectConfigPath(cfg.ProjectRoot)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return fmt.Errorf("no config file found at %s", cfgPath)
	}

	f, err := readConfigFile(cfgPath)
	if err != nil {
		return err
	}

	cs := getConfigStruct(f)
	if cs == nil {
		return fmt.Errorf("no Config block found in %s", cfgPath)
	}

	cols := findFieldInStruct(cs, "collections")
	if cols == nil {
		return fmt.Errorf("no collections found in config")
	}
	colSt, ok := cols.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("collections is not a struct")
	}

	found := false
	for _, d := range colSt.Elts {
		if fld, ok := d.(*ast.Field); ok && fieldLabel(fld) == oldName {
			fld.Label = ast.NewIdent(newName)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("collection %q not found in config", oldName)
	}

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	cfg.Collections[newName] = cfg.Collections[oldName]
	delete(cfg.Collections, oldName)
	return nil
}

// AddCollectionPatterns adds patterns to a collection.
// If replaceAll is true, existing patterns are replaced entirely.
func AddCollectionPatterns(cfg *Config, name string, patterns []string, replaceAll bool) error {
	if _, exists := cfg.Collections[name]; !exists {
		return fmt.Errorf("collection %q not found", name)
	}

	cfgPath := ProjectConfigPath(cfg.ProjectRoot)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return fmt.Errorf("no config file found at %s", cfgPath)
	}

	f, err := readConfigFile(cfgPath)
	if err != nil {
		return err
	}

	cs := getConfigStruct(f)
	if cs == nil {
		return fmt.Errorf("no Config block found in %s", cfgPath)
	}

	cols := findFieldInStruct(cs, "collections")
	if cols == nil {
		return fmt.Errorf("no collections found in config")
	}
	colSt, ok := cols.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("collections is not a struct")
	}

	colField := findFieldInStruct(colSt, name)
	if colField == nil {
		return fmt.Errorf("collection %q not found in config", name)
	}
	colInner, ok := colField.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("collection %q is not a struct", name)
	}

	if replaceAll {
		if existing := findFieldInStruct(colInner, "patterns"); existing != nil {
			list := &ast.ListLit{}
			for _, p := range patterns {
				list.Elts = append(list.Elts, ast.NewString(p))
			}
			existing.Value = list
		} else {
			colInner.Elts = append(colInner.Elts, strListField("patterns", patterns))
		}
		col := cfg.Collections[name]
		col.Patterns = patterns
		cfg.Collections[name] = col
	} else {
		existing := findFieldInStruct(colInner, "patterns")
		if existing != nil {
			if list, ok := existing.Value.(*ast.ListLit); ok {
				seen := make(map[string]bool, len(list.Elts))
				for _, elt := range list.Elts {
					if lit, ok := elt.(*ast.BasicLit); ok {
						seen[lit.Value] = true
					}
				}
				for _, p := range patterns {
					q := fmt.Sprintf("%q", p)
					if !seen[q] {
						list.Elts = append(list.Elts, ast.NewString(p))
						seen[q] = true
					}
				}
			}
		} else {
			list := &ast.ListLit{}
			for _, p := range patterns {
				list.Elts = append(list.Elts, ast.NewString(p))
			}
			colInner.Elts = append(colInner.Elts, &ast.Field{
				Label: ast.NewIdent("patterns"),
				Value: list,
			})
		}
		col := cfg.Collections[name]
		seen := make(map[string]bool, len(col.Patterns))
		for _, p := range col.Patterns {
			seen[p] = true
		}
		for _, p := range patterns {
			if !seen[p] {
				col.Patterns = append(col.Patterns, p)
				seen[p] = true
			}
		}
		cfg.Collections[name] = col
	}

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	return nil
}

// AddIgnorePatterns adds ignore patterns to a collection.
// If replaceAll is true, existing ignore patterns are replaced entirely.
func AddIgnorePatterns(cfg *Config, name string, patterns []string, replaceAll bool) error {
	if _, exists := cfg.Collections[name]; !exists {
		return fmt.Errorf("collection %q not found", name)
	}

	cfgPath := ProjectConfigPath(cfg.ProjectRoot)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return fmt.Errorf("no config file found at %s", cfgPath)
	}

	f, err := readConfigFile(cfgPath)
	if err != nil {
		return err
	}

	cs := getConfigStruct(f)
	if cs == nil {
		return fmt.Errorf("no Config block found in %s", cfgPath)
	}

	cols := findFieldInStruct(cs, "collections")
	if cols == nil {
		return fmt.Errorf("no collections found in config")
	}
	colSt, ok := cols.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("collections is not a struct")
	}

	colField := findFieldInStruct(colSt, name)
	if colField == nil {
		return fmt.Errorf("collection %q not found in config", name)
	}
	colInner, ok := colField.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("collection %q is not a struct", name)
	}

	if replaceAll {
		if existing := findFieldInStruct(colInner, "ignore"); existing != nil {
			list := &ast.ListLit{}
			for _, p := range patterns {
				list.Elts = append(list.Elts, ast.NewString(p))
			}
			existing.Value = list
		} else {
			colInner.Elts = append(colInner.Elts, strListField("ignore", patterns))
		}
		col := cfg.Collections[name]
		col.Ignore = patterns
		cfg.Collections[name] = col
	} else {
		existing := findFieldInStruct(colInner, "ignore")
		if existing != nil {
			if list, ok := existing.Value.(*ast.ListLit); ok {
				seen := make(map[string]bool, len(list.Elts))
				for _, elt := range list.Elts {
					if lit, ok := elt.(*ast.BasicLit); ok {
						seen[lit.Value] = true
					}
				}
				for _, p := range patterns {
					q := fmt.Sprintf("%q", p)
					if !seen[q] {
						list.Elts = append(list.Elts, ast.NewString(p))
						seen[q] = true
					}
				}
			}
		} else {
			list := &ast.ListLit{}
			for _, p := range patterns {
				list.Elts = append(list.Elts, ast.NewString(p))
			}
			colInner.Elts = append(colInner.Elts, &ast.Field{
				Label: ast.NewIdent("ignore"),
				Value: list,
			})
		}
		col := cfg.Collections[name]
		seen := make(map[string]bool, len(col.Ignore))
		for _, ig := range col.Ignore {
			seen[ig] = true
		}
		for _, p := range patterns {
			if !seen[p] {
				col.Ignore = append(col.Ignore, p)
				seen[p] = true
			}
		}
		cfg.Collections[name] = col
	}

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	return nil
}

// RemoveIgnorePattern removes an ignore pattern from a collection.
func RemoveIgnorePattern(cfg *Config, name, pattern string) error {
	if _, exists := cfg.Collections[name]; !exists {
		return fmt.Errorf("collection %q not found", name)
	}

	cfgPath := ProjectConfigPath(cfg.ProjectRoot)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return fmt.Errorf("no config file found at %s", cfgPath)
	}

	f, err := readConfigFile(cfgPath)
	if err != nil {
		return err
	}

	cs := getConfigStruct(f)
	if cs == nil {
		return fmt.Errorf("no Config block found in %s", cfgPath)
	}

	cols := findFieldInStruct(cs, "collections")
	if cols == nil {
		return fmt.Errorf("no collections found in config")
	}
	colSt, ok := cols.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("collections is not a struct")
	}

	colField := findFieldInStruct(colSt, name)
	if colField == nil {
		return fmt.Errorf("collection %q not found in config", name)
	}
	colInner, ok := colField.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("collection %q is not a struct", name)
	}

	existing := findFieldInStruct(colInner, "ignore")
	if existing == nil {
		return nil
	}
	list, ok := existing.Value.(*ast.ListLit)
	if !ok {
		return fmt.Errorf("ignore field is not a list")
	}

	quoted := fmt.Sprintf("%q", pattern)
	for i, elt := range list.Elts {
		if lit, ok := elt.(*ast.BasicLit); ok && lit.Value == quoted {
			list.Elts = append(list.Elts[:i], list.Elts[i+1:]...)
			break
		}
	}

	if len(list.Elts) == 0 {
		// Remove the entire ignore field
		for i, d := range colInner.Elts {
			if fld, ok := d.(*ast.Field); ok && fieldLabel(fld) == "ignore" {
				colInner.Elts = append(colInner.Elts[:i], colInner.Elts[i+1:]...)
				break
			}
		}
	}

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	col := cfg.Collections[name]
	newIgnore := make([]string, 0, len(col.Ignore))
	for _, ig := range col.Ignore {
		if ig != pattern {
			newIgnore = append(newIgnore, ig)
		}
	}
	col.Ignore = newIgnore
	cfg.Collections[name] = col
	return nil
}

// AddContextDoc adds a context document to a collection.
func AddContextDoc(cfg *Config, name, ctxPath string) error {
	if _, exists := cfg.Collections[name]; !exists {
		return fmt.Errorf("collection %q not found", name)
	}

	cfgPath := ProjectConfigPath(cfg.ProjectRoot)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return fmt.Errorf("no config file found at %s", cfgPath)
	}

	f, err := readConfigFile(cfgPath)
	if err != nil {
		return err
	}

	cs := getConfigStruct(f)
	if cs == nil {
		return fmt.Errorf("no Config block found in %s", cfgPath)
	}

	cols := findFieldInStruct(cs, "collections")
	if cols == nil {
		return fmt.Errorf("no collections found in config")
	}
	colSt, ok := cols.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("collections is not a struct")
	}

	colField := findFieldInStruct(colSt, name)
	if colField == nil {
		return fmt.Errorf("collection %q not found in config", name)
	}
	colInner, ok := colField.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("collection %q is not a struct", name)
	}

	if existing := findFieldInStruct(colInner, "context"); existing != nil {
		existing.Value = ast.NewString(ctxPath)
	} else {
		colInner.Elts = append(colInner.Elts, strField("context", ctxPath))
	}

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	col := cfg.Collections[name]
	col.Context = ctxPath
	cfg.Collections[name] = col
	return nil
}

// RemoveContextDoc removes the context document from a collection.
func RemoveContextDoc(cfg *Config, name string) error {
	if _, exists := cfg.Collections[name]; !exists {
		return fmt.Errorf("collection %q not found", name)
	}

	cfgPath := ProjectConfigPath(cfg.ProjectRoot)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return fmt.Errorf("no config file found at %s", cfgPath)
	}

	f, err := readConfigFile(cfgPath)
	if err != nil {
		return err
	}

	cs := getConfigStruct(f)
	if cs == nil {
		return fmt.Errorf("no Config block found in %s", cfgPath)
	}

	cols := findFieldInStruct(cs, "collections")
	if cols == nil {
		return fmt.Errorf("no collections found in config")
	}
	colSt, ok := cols.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("collections is not a struct")
	}

	colField := findFieldInStruct(colSt, name)
	if colField == nil {
		return fmt.Errorf("collection %q not found in config", name)
	}
	colInner, ok := colField.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("collection %q is not a struct", name)
	}

	found := false
	for i, d := range colInner.Elts {
		if fld, ok := d.(*ast.Field); ok && fieldLabel(fld) == "context" {
			colInner.Elts = append(colInner.Elts[:i], colInner.Elts[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	col := cfg.Collections[name]
	col.Context = ""
	cfg.Collections[name] = col
	return nil
}

// ListContextDocs returns a map of collection name to context document path.
func ListContextDocs(cfg *Config) map[string]string {
	result := make(map[string]string)
	for name, col := range cfg.Collections {
		if col.Context != "" {
			result[name] = col.Context
		}
	}
	return result
}
