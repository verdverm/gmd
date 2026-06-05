package wiki

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/verdverm/gmd/pkg/config"
)

var wikiDirs = []string{
	"raw",
	"wiki/entities",
	"wiki/concepts",
	"wiki/comparisons",
	"wiki/synthesis",
	"wiki/sources",
}

type Wiki struct {
	Name       string
	Path       string
	WikiPath   string
	RawPath    string
	Config     *config.WikiConfig
	WikiConfig *config.WikiConfig
}

func NewWiki(name string, indexPath string, cfg *config.WikiConfig) (*Wiki, error) {
	return &Wiki{
		Name:       name,
		Path:       indexPath,
		WikiPath:   filepath.Join(indexPath, cfg.WikiDir),
		RawPath:    filepath.Join(indexPath, cfg.RawDir),
		Config:     cfg,
		WikiConfig: cfg,
	}, nil
}

func InitWiki(name, wikiPath string, cfg *config.WikiConfig) error {
	w, err := NewWiki(name, wikiPath, cfg)
	if err != nil {
		return err
	}
	return w.Init()
}

func (w *Wiki) Init() error {
	for _, dir := range wikiDirs {
		p := filepath.Join(w.Path, dir)
		if err := os.MkdirAll(p, 0755); err != nil {
			return fmt.Errorf("creating %s: %w", p, err)
		}
	}

	indexPath := filepath.Join(w.WikiPath, w.WikiConfig.IndexFile)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		content := "# Wiki Index\n\n## Entities\n\n## Concepts\n\n## Comparisons\n\n## Sources\n\n## Last Updated\n\n"
		if err := os.WriteFile(indexPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing index file: %w", err)
		}
	}

	logPath := filepath.Join(w.WikiPath, w.WikiConfig.LogFile)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		content := "# Wiki Log\n\n"
		if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing log file: %w", err)
		}
	}

	schemaPath := filepath.Join(w.Path, "WIKI_SCHEMA.md")
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		tmpl, err := GetSkillTemplate("WIKI_SCHEMA.md")
		if err == nil {
			if err := os.WriteFile(schemaPath, []byte(tmpl.Content), 0644); err != nil {
				return fmt.Errorf("writing schema file: %w", err)
			}
		}
	}

	return nil
}

func (w *Wiki) IndexFilePath() string {
	return filepath.Join(w.WikiPath, w.WikiConfig.IndexFile)
}

func (w *Wiki) LogFilePath() string {
	return filepath.Join(w.WikiPath, w.WikiConfig.LogFile)
}
