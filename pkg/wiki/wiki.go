package wiki

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/verdverm/gmd/pkg/config"
)

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
	rawPath := filepath.Join(w.Path, w.WikiConfig.RawDir)
	if err := os.MkdirAll(rawPath, 0755); err != nil {
		return fmt.Errorf("creating %s: %w", rawPath, err)
	}

	if err := os.MkdirAll(w.WikiPath, 0755); err != nil {
		return fmt.Errorf("creating %s: %w", w.WikiPath, err)
	}

	indexPath := filepath.Join(w.WikiPath, w.WikiConfig.IndexFile)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		content := fmt.Sprintf("---\nokf_version: \"%s\"\n---\n\n# Wiki Index\n\n## Last Updated\n\n", w.WikiConfig.OkfVersion)
		if err := os.WriteFile(indexPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("writing index file: %w", err)
		}
	}

	logPath := filepath.Join(w.WikiPath, w.WikiConfig.LogFile)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		content := "# Wiki Log\n\n"
		if err := os.WriteFile(logPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("writing log file: %w", err)
		}
	}

	schemaPath := filepath.Join(w.Path, "WIKI_SCHEMA.md")
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		data, err := wikiEmbedsFS.ReadFile("embeds/wiki_schema.md")
		if err == nil {
			if err := os.WriteFile(schemaPath, data, 0600); err != nil {
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
