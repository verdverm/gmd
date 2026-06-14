package wiki

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/verdverm/gmd/pkg/chunking"
)

type OKFViolation struct {
	Page    string
	Message string
	IsError bool
}

type OKFReport struct {
	Violations []OKFViolation
	PassCount  int
	ErrorCount int
}

func (r *OKFReport) HasErrors() bool {
	return r.ErrorCount > 0
}

var ErrOKFValidation = errors.New("OKF validation failed")

func ValidateOKF(wiki *Wiki) (*OKFReport, error) {
	report := &OKFReport{}
	wikiDir := wiki.WikiPath
	indexFile := wiki.WikiConfig.IndexFile
	logFile := wiki.WikiConfig.LogFile

	err := filepath.Walk(wikiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		base := filepath.Base(path)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			report.Violations = append(report.Violations, OKFViolation{
				Page:    pageName(wikiDir, path),
				Message: fmt.Sprintf("cannot read: %v", readErr),
				IsError: true,
			})
			report.ErrorCount++
			return nil
		}

		fm, _, parseErr := ParseFrontmatter(string(data))
		if parseErr != nil {
			report.Violations = append(report.Violations, OKFViolation{
				Page:    pageName(wikiDir, path),
				Message: fmt.Sprintf("YAML frontmatter parse error: %v", parseErr),
				IsError: true,
			})
			report.ErrorCount++
			return nil
		}

		if base == indexFile {
			rel, _ := filepath.Rel(wikiDir, path)
			if rel != indexFile {
				// Subdirectory index.md must have NO frontmatter (OKF §6)
				if fm != nil {
					report.Violations = append(report.Violations, OKFViolation{
						Page:    pageName(wikiDir, path),
						Message: fmt.Sprintf("subdirectory %s must have no frontmatter (OKF §6); only bundle-root %s may have okf_version", base, indexFile),
						IsError: true,
					})
					report.ErrorCount++
				} else {
					report.PassCount++
				}
			}
			// Root index.md with frontmatter is fine (okf_version)
		} else if base == logFile {
			// log.md has no frontmatter requirement
		} else {
			if fm == nil {
				report.Violations = append(report.Violations, OKFViolation{
					Page:    pageName(wikiDir, path),
					Message: "missing YAML frontmatter (OKF §4.1 requires frontmatter on every .md file)",
					IsError: true,
				})
				report.ErrorCount++
			} else {
				t, ok := fm["type"]
				if !ok || t == nil || t == "" {
					report.Violations = append(report.Violations, OKFViolation{
						Page:    pageName(wikiDir, path),
						Message: "missing required frontmatter field 'type' (OKF §4.1)",
						IsError: true,
					})
					report.ErrorCount++
				}
			}
		}

		report.PassCount++
		return nil
	})

	if err != nil {
		return report, fmt.Errorf("walking wiki directory: %w", err)
	}

	return report, nil
}

func ExportOKF(wiki *Wiki, outputDir string) (*OKFReport, error) {
	report := &OKFReport{}
	wikiDir := wiki.WikiPath

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return report, fmt.Errorf("creating output directory: %w", err)
	}

	pageNameToID := buildPageRegistry(wikiDir)

	err := filepath.Walk(wikiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		rel, relErr := filepath.Rel(wikiDir, path)
		if relErr != nil {
			return nil
		}

		destPath := filepath.Join(outputDir, rel)
		destDir := filepath.Dir(destPath)
		if mkdirErr := os.MkdirAll(destDir, 0755); mkdirErr != nil {
			return mkdirErr
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		content := string(data)

		fm, stripped, fmErr := ParseFrontmatter(content)
		if fmErr != nil {
			return writeExportFile(destPath, content)
		}

		if filepath.Base(rel) == wiki.WikiConfig.IndexFile {
			if _, ok := fm["okf_version"]; !ok {
				fm["okf_version"] = wiki.WikiConfig.OkfVersion
			}
		}

		converted := convertWikilinksToMarkdown(stripped, filepath.Dir(rel), pageNameToID)

		fmYAML, _ := marshalYAML(fm)
		exportContent := fmt.Sprintf("---\n%s\n---\n\n%s", fmYAML, converted)
		if err := writeExportFile(destPath, exportContent); err != nil {
			return err
		}

		report.PassCount++
		return nil
	})

	if err != nil {
		return report, err
	}

	return report, nil
}

func buildPageRegistry(wikiDir string) map[string]string {
	registry := make(map[string]string)
	_ = filepath.Walk(wikiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		cid := pageName(wikiDir, path)
		data, _ := os.ReadFile(path)
		fm, stripped, _ := ParseFrontmatter(string(data))
		pn := getPageName(fm, stripped)
		if pn != "" {
			registry[pn] = cid
		}
		return nil
	})
	return registry
}

func convertWikilinksToMarkdown(content string, sourceRelDir string, pageNameToID map[string]string) string {
	wikilinks := chunking.ExtractWikilinks(content)
	result := content
	for _, link := range wikilinks {
		var targetPath string
		if id, ok := pageNameToID[link]; ok {
			targetPath = "/" + id + ".md"
		} else {
			targetPath = "/" + link + ".md"
		}
		oldPattern := "[[" + link + "]]"
		newLink := "[" + link + "](" + targetPath + ")"
		result = strings.ReplaceAll(result, oldPattern, newLink)
		oldPatAlias := "[[" + link + "|"
		if idx := strings.Index(result, oldPatAlias); idx >= 0 {
			end := strings.Index(result[idx:], "]]")
			if end >= 0 {
				alias := result[idx+len(oldPatAlias) : idx+end]
				newLinkAlias := "[" + alias + "](" + targetPath + ")"
				result = result[:idx] + newLinkAlias + result[idx+end+2:]
			}
		}
	}
	return result
}

func writeExportFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0600)
}

var _ = time.Now
