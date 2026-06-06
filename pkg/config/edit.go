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

// sourceMapField resolves a name to the correct CUE map ("collections" or "wikis")
// and returns the struct field containing the inner struct. Returns "", nil if not found.
func sourceMapField(cfg *Config, cs *ast.StructLit, name string) (string, *ast.Field, *ast.StructLit) {
	// Check collections
	if _, ok := cfg.Collections[name]; ok {
		cols := findFieldInStruct(cs, "collections")
		if cols != nil {
			if colSt, ok := cols.Value.(*ast.StructLit); ok {
				if f := findFieldInStruct(colSt, name); f != nil {
					return "collections", cols, colSt
				}
			}
		}
	}
	// Check wikis
	if _, ok := cfg.Wikis[name]; ok {
		wikis := findFieldInStruct(cs, "wikis")
		if wikis != nil {
			if wikiSt, ok := wikis.Value.(*ast.StructLit); ok {
				if f := findFieldInStruct(wikiSt, name); f != nil {
					return "wikis", wikis, wikiSt
				}
			}
		}
	}
	return "", nil, nil
}

// getSourceInner returns the AST inner struct for a named source (collection or wiki).
func getSourceInner(cfg *Config, cs *ast.StructLit, name string) (*ast.StructLit, error) {
	_, _, parentSt := sourceMapField(cfg, cs, name)
	if parentSt == nil {
		return nil, fmt.Errorf("source %q not found in config", name)
	}
	colField := findFieldInStruct(parentSt, name)
	if colField == nil {
		return nil, fmt.Errorf("source %q not found in config", name)
	}
	colInner, ok := colField.Value.(*ast.StructLit)
	if !ok {
		return nil, fmt.Errorf("source %q is not a struct", name)
	}
	return colInner, nil
}

// AddCollection adds a new collection to the project config file.
func AddCollection(cfg *Config, name, path string, patterns []string) error {
	if cfg.SourceExists(name) {
		return fmt.Errorf("source %q already exists", name)
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
	if cfg.Collections == nil {
		cfg.Collections = make(map[string]CollectionConfig)
	}
	cfg.Collections[name] = CollectionConfig{
		SourceConfig: SourceConfig{
			Path:     path,
			Patterns: patterns,
		},
	}

	return nil
}

// RemoveCollection removes a collection from the project config file and
// returns the removed collection config.
func RemoveCollection(cfg *Config, name string) error {
	if !cfg.SourceExists(name) {
		return fmt.Errorf("source %q not found", name)
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

	mapKey, _, parentSt := sourceMapField(cfg, cs, name)
	if parentSt == nil {
		return fmt.Errorf("source %q not found in config", name)
	}

	found := false
	for i, d := range parentSt.Elts {
		if fld, ok := d.(*ast.Field); ok && fieldLabel(fld) == name {
			parentSt.Elts = append(parentSt.Elts[:i], parentSt.Elts[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("source %q not found in config", name)
	}

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	switch mapKey {
	case "collections":
		delete(cfg.Collections, name)
	case "wikis":
		delete(cfg.Wikis, name)
	}
	return nil
}

// RenameCollection renames a collection in the project config file.
func RenameCollection(cfg *Config, oldName, newName string) error {
	if !cfg.SourceExists(oldName) {
		return fmt.Errorf("source %q not found", oldName)
	}
	if cfg.SourceExists(newName) {
		return fmt.Errorf("source %q already exists", newName)
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

	mapKey, _, parentSt := sourceMapField(cfg, cs, oldName)
	if parentSt == nil {
		return fmt.Errorf("source %q not found in config", oldName)
	}

	found := false
	for _, d := range parentSt.Elts {
		if fld, ok := d.(*ast.Field); ok && fieldLabel(fld) == oldName {
			fld.Label = ast.NewIdent(newName)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("source %q not found in config", oldName)
	}

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	switch mapKey {
	case "collections":
		cfg.Collections[newName] = cfg.Collections[oldName]
		delete(cfg.Collections, oldName)
	case "wikis":
		cfg.Wikis[newName] = cfg.Wikis[oldName]
		delete(cfg.Wikis, oldName)
	}
	return nil
}

// AddCollectionPatterns adds patterns to a source (collection or wiki).
// If replaceAll is true, existing patterns are replaced entirely.
func AddCollectionPatterns(cfg *Config, name string, patterns []string, replaceAll bool) error {
	if !cfg.SourceExists(name) {
		return fmt.Errorf("source %q not found", name)
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

	colInner, err := getSourceInner(cfg, cs, name)
	if err != nil {
		return err
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
		updateSourcePatterns(cfg, name, patterns)
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
		appendSourcePatterns(cfg, name, patterns)
	}

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	return nil
}

func updateSourcePatterns(cfg *Config, name string, patterns []string) {
	if col, ok := cfg.Collections[name]; ok {
		col.Patterns = patterns
		cfg.Collections[name] = col
	} else if wc, ok := cfg.Wikis[name]; ok {
		wc.Patterns = patterns
		cfg.Wikis[name] = wc
	}
}

func appendSourcePatterns(cfg *Config, name string, patterns []string) {
	if col, ok := cfg.Collections[name]; ok {
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
	} else if wc, ok := cfg.Wikis[name]; ok {
		seen := make(map[string]bool, len(wc.Patterns))
		for _, p := range wc.Patterns {
			seen[p] = true
		}
		for _, p := range patterns {
			if !seen[p] {
				wc.Patterns = append(wc.Patterns, p)
				seen[p] = true
			}
		}
		cfg.Wikis[name] = wc
	}
}

// AddIgnorePatterns adds ignore patterns to a source (collection or wiki).
// If replaceAll is true, existing ignore patterns are replaced entirely.
func AddIgnorePatterns(cfg *Config, name string, patterns []string, replaceAll bool) error {
	if !cfg.SourceExists(name) {
		return fmt.Errorf("source %q not found", name)
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

	colInner, err := getSourceInner(cfg, cs, name)
	if err != nil {
		return err
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
		updateSourceIgnore(cfg, name, patterns)
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
		appendSourceIgnore(cfg, name, patterns)
	}

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	return nil
}

func updateSourceIgnore(cfg *Config, name string, patterns []string) {
	if col, ok := cfg.Collections[name]; ok {
		col.Ignore = patterns
		cfg.Collections[name] = col
	} else if wc, ok := cfg.Wikis[name]; ok {
		wc.Ignore = patterns
		cfg.Wikis[name] = wc
	}
}

func appendSourceIgnore(cfg *Config, name string, patterns []string) {
	if col, ok := cfg.Collections[name]; ok {
		seen := make(map[string]bool, len(col.Ignore))
		for _, p := range col.Ignore {
			seen[p] = true
		}
		for _, p := range patterns {
			if !seen[p] {
				col.Ignore = append(col.Ignore, p)
				seen[p] = true
			}
		}
		cfg.Collections[name] = col
	} else if wc, ok := cfg.Wikis[name]; ok {
		seen := make(map[string]bool, len(wc.Ignore))
		for _, p := range wc.Ignore {
			seen[p] = true
		}
		for _, p := range patterns {
			if !seen[p] {
				wc.Ignore = append(wc.Ignore, p)
				seen[p] = true
			}
		}
		cfg.Wikis[name] = wc
	}
}

// RemoveIgnorePattern removes an ignore pattern from a source (collection or wiki).
func RemoveIgnorePattern(cfg *Config, name, pattern string) error {
	if !cfg.SourceExists(name) {
		return fmt.Errorf("source %q not found", name)
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

	colInner, err := getSourceInner(cfg, cs, name)
	if err != nil {
		return err
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

	removeSourceIgnore(cfg, name, pattern)
	return nil
}

func removeSourceIgnore(cfg *Config, name, pattern string) {
	if col, ok := cfg.Collections[name]; ok {
		newIgnore := make([]string, 0, len(col.Ignore))
		for _, ig := range col.Ignore {
			if ig != pattern {
				newIgnore = append(newIgnore, ig)
			}
		}
		col.Ignore = newIgnore
		cfg.Collections[name] = col
	} else if wc, ok := cfg.Wikis[name]; ok {
		newIgnore := make([]string, 0, len(wc.Ignore))
		for _, ig := range wc.Ignore {
			if ig != pattern {
				newIgnore = append(newIgnore, ig)
			}
		}
		wc.Ignore = newIgnore
		cfg.Wikis[name] = wc
	}
}

// AddContextDoc adds a context document to a source (collection or wiki).
func AddContextDoc(cfg *Config, name, ctxPath string) error {
	if !cfg.SourceExists(name) {
		return fmt.Errorf("source %q not found", name)
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

	colInner, err := getSourceInner(cfg, cs, name)
	if err != nil {
		return err
	}

	if existing := findFieldInStruct(colInner, "context"); existing != nil {
		existing.Value = ast.NewString(ctxPath)
	} else {
		colInner.Elts = append(colInner.Elts, strField("context", ctxPath))
	}

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	if col, ok := cfg.Collections[name]; ok {
		col.Context = ctxPath
		cfg.Collections[name] = col
	} else if wc, ok := cfg.Wikis[name]; ok {
		wc.Context = ctxPath
		cfg.Wikis[name] = wc
	}
	return nil
}

// RemoveContextDoc removes the context document from a source (collection or wiki).
func RemoveContextDoc(cfg *Config, name string) error {
	if !cfg.SourceExists(name) {
		return fmt.Errorf("source %q not found", name)
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

	colInner, err := getSourceInner(cfg, cs, name)
	if err != nil {
		return err
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

	if col, ok := cfg.Collections[name]; ok {
		col.Context = ""
		cfg.Collections[name] = col
	} else if wc, ok := cfg.Wikis[name]; ok {
		wc.Context = ""
		cfg.Wikis[name] = wc
	}
	return nil
}

// ListContextDocs returns a map of source name to context document path.
func ListContextDocs(cfg *Config) map[string]string {
	result := make(map[string]string)
	for name, col := range cfg.Collections {
		if col.Context != "" {
			result[name] = col.Context
		}
	}
	for name, wc := range cfg.Wikis {
		if wc.Context != "" {
			result[name] = wc.Context
		}
	}
	return result
}

// CreateWiki adds a new wiki to the project config file and initializes
// in-memory config. Validates name uniqueness but not path conflicts or cycles.
func CreateWiki(cfg *Config, name, path string, patterns []string, wikiDir, rawDir string, sourceRefs []string) error {
	if cfg.SourceExists(name) {
		return fmt.Errorf("source %q already exists", name)
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

	wikis := getOrCreateStruct(cs, "wikis")
	if findFieldInStruct(wikis, name) != nil {
		return fmt.Errorf("wiki %q already exists in config", name)
	}

	wikiStruct := &ast.StructLit{}
	wikiStruct.Elts = append(wikiStruct.Elts,
		strField("path", path),
		strListField("patterns", patterns),
	)
	if wikiDir != "" && wikiDir != "wiki" {
		wikiStruct.Elts = append(wikiStruct.Elts, strField("wikiDir", wikiDir))
	}
	if rawDir != "" && rawDir != "raw" {
		wikiStruct.Elts = append(wikiStruct.Elts, strField("rawDir", rawDir))
	}
	if len(sourceRefs) > 0 {
		wikiStruct.Elts = append(wikiStruct.Elts, strListField("sourceRefs", sourceRefs))
	}

	wikis.Elts = append(wikis.Elts, &ast.Field{
		Label: ast.NewIdent(name),
		Value: wikiStruct,
	})

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	// Update in-memory config
	if cfg.Wikis == nil {
		cfg.Wikis = make(map[string]WikiConfig)
	}
	wd := wikiDir
	if wd == "" {
		wd = "wiki"
	}
	rd := rawDir
	if rd == "" {
		rd = "raw"
	}
	cfg.Wikis[name] = WikiConfig{
		SourceConfig: SourceConfig{
			Path:     path,
			Patterns: patterns,
		},
		WikiDir:    wd,
		RawDir:     rd,
		SourceRefs: sourceRefs,
	}

	return nil
}

// AddSourceRef adds a source reference to a wiki. Validates target exists and
// checks for cycles.
func AddSourceRef(cfg *Config, wikiName, srcName string) error {
	wc, ok := cfg.Wikis[wikiName]
	if !ok {
		return fmt.Errorf("wiki %q not found", wikiName)
	}

	if srcName == wikiName {
		return fmt.Errorf("wiki cannot reference itself")
	}

	if !cfg.SourceExists(srcName) {
		return fmt.Errorf("source %q not found in collections or wikis", srcName)
	}

	// Check for cycles (only relevant when src is a wiki)
	if _, isWiki := cfg.Wikis[srcName]; isWiki {
		if cfg.WouldCreateSourceRefsCycle(wikiName, srcName) {
			return fmt.Errorf("adding source ref %q to wiki %q would create a circular reference", srcName, wikiName)
		}
	}

	// Dedup
	for _, ref := range wc.SourceRefs {
		if ref == srcName {
			return nil // already added
		}
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

	wikis := findFieldInStruct(cs, "wikis")
	if wikis == nil {
		return fmt.Errorf("no wikis found in config")
	}
	wikisSt, ok := wikis.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("wikis is not a struct")
	}
	wikiField := findFieldInStruct(wikisSt, wikiName)
	if wikiField == nil {
		return fmt.Errorf("wiki %q not found in config", wikiName)
	}
	wikiInner, ok := wikiField.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("wiki %q is not a struct", wikiName)
	}

	existing := findFieldInStruct(wikiInner, "sourceRefs")
	if existing != nil {
		if list, ok := existing.Value.(*ast.ListLit); ok {
			list.Elts = append(list.Elts, ast.NewString(srcName))
		}
	} else {
		wikiInner.Elts = append(wikiInner.Elts, strListField("sourceRefs", []string{srcName}))
	}

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	wc.SourceRefs = append(wc.SourceRefs, srcName)
	cfg.Wikis[wikiName] = wc
	return nil
}

// RemoveSourceRef removes a source reference from a wiki.
func RemoveSourceRef(cfg *Config, wikiName, srcName string) error {
	wc, ok := cfg.Wikis[wikiName]
	if !ok {
		return fmt.Errorf("wiki %q not found", wikiName)
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

	wikis := findFieldInStruct(cs, "wikis")
	if wikis == nil {
		return fmt.Errorf("no wikis found in config")
	}
	wikisSt, ok := wikis.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("wikis is not a struct")
	}
	wikiField := findFieldInStruct(wikisSt, wikiName)
	if wikiField == nil {
		return fmt.Errorf("wiki %q not found in config", wikiName)
	}
	wikiInner, ok := wikiField.Value.(*ast.StructLit)
	if !ok {
		return fmt.Errorf("wiki %q is not a struct", wikiName)
	}

	existing := findFieldInStruct(wikiInner, "sourceRefs")
	if existing != nil {
		if list, ok := existing.Value.(*ast.ListLit); ok {
			quoted := fmt.Sprintf("%q", srcName)
			for i, elt := range list.Elts {
				if lit, ok := elt.(*ast.BasicLit); ok && lit.Value == quoted {
					list.Elts = append(list.Elts[:i], list.Elts[i+1:]...)
					break
				}
			}
			if len(list.Elts) == 0 {
				for i, d := range wikiInner.Elts {
					if fld, ok := d.(*ast.Field); ok && fieldLabel(fld) == "sourceRefs" {
						wikiInner.Elts = append(wikiInner.Elts[:i], wikiInner.Elts[i+1:]...)
						break
					}
				}
			}
		}
	}

	if err := writeConfigFile(cfgPath, f); err != nil {
		return err
	}

	newRefs := make([]string, 0, len(wc.SourceRefs))
	for _, ref := range wc.SourceRefs {
		if ref != srcName {
			newRefs = append(newRefs, ref)
		}
	}
	wc.SourceRefs = newRefs
	cfg.Wikis[wikiName] = wc
	return nil
}
