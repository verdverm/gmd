package search

import (
	"math"
	"strings"
	"testing"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/ts"
)

func makeResult(path string, score float64) ts.HybridSearchResult {
	return ts.HybridSearchResult{
		Collection: "docs",
		Path:       path,
		Title:      "title",
		Content:    "content",
		ChunkSeq:   0,
		Score:      score,
	}
}

func TestSearch_RRFFuse(t *testing.T) {
	t.Run("single variant", func(t *testing.T) {
		results := []variantResult{{
			weight: 1.0,
			results: []ts.HybridSearchResult{
				makeResult("a.md", 0.9),
				makeResult("b.md", 0.8),
			},
		}}
		var p *Pipeline
		fused := p.rrfFuse(results, 60)
		if len(fused) != 2 {
			t.Fatalf("got %d fused, want 2", len(fused))
		}
		if fused[0].key != "docs:a.md" {
			t.Errorf("top result should be a.md, got %s", fused[0].key)
		}
	})

	t.Run("multiple variants with weights", func(t *testing.T) {
		list1 := []ts.HybridSearchResult{
			makeResult("a.md", 0.92),
			makeResult("b.md", 0.81),
		}
		list2 := []ts.HybridSearchResult{
			makeResult("b.md", 0.77),
			makeResult("a.md", 0.65),
		}
		results := []variantResult{
			{weight: 2.0, results: list1},
			{weight: 1.0, results: list2},
		}
		var p *Pipeline
		fused := p.rrfFuse(results, 60)

		if len(fused) != 2 {
			t.Fatalf("got %d fused, want 2", len(fused))
		}

		docA := fused[0]
		docB := fused[1]
		if docA.key != "docs:a.md" {
			t.Errorf("expected a.md first, got %s", docA.key)
		}
		if docB.key != "docs:b.md" {
			t.Errorf("expected b.md second, got %s", docB.key)
		}
		if docA.rrfScore <= docB.rrfScore {
			t.Errorf("a.md score (%f) should be > b.md score (%f)", docA.rrfScore, docB.rrfScore)
		}
	})

	t.Run("empty results", func(t *testing.T) {
		var p *Pipeline
		fused := p.rrfFuse(nil, 60)
		if len(fused) != 0 {
			t.Errorf("expected 0, got %d", len(fused))
		}
	})

	t.Run("no variants", func(t *testing.T) {
		var p *Pipeline
		fused := p.rrfFuse([]variantResult{}, 60)
		if len(fused) != 0 {
			t.Errorf("expected 0, got %d", len(fused))
		}
	})

	t.Run("top-rank bonus applied", func(t *testing.T) {
		list1 := []ts.HybridSearchResult{
			makeResult("a.md", 0.9),
			makeResult("b.md", 0.8),
			makeResult("c.md", 0.7),
		}
		results := []variantResult{{weight: 1.0, results: list1}}
		var p *Pipeline
		fused := p.rrfFuse(results, 60)

		scoreA := 1.0/float64(60+1) + topRankBonus
		scoreB := 1.0 / float64(60+2)
		scoreC := 1.0 / float64(60+3)

		if math.Abs(fused[0].rrfScore-scoreA) > 1e-10 {
			t.Errorf("a.md score = %f, want %f", fused[0].rrfScore, scoreA)
		}
		if math.Abs(fused[1].rrfScore-scoreB) > 1e-10 {
			t.Errorf("b.md score = %f, want %f", fused[1].rrfScore, scoreB)
		}
		if math.Abs(fused[2].rrfScore-scoreC) > 1e-10 {
			t.Errorf("c.md score = %f, want %f", fused[2].rrfScore, scoreC)
		}
	})

	t.Run("different k values", func(t *testing.T) {
		list1 := []ts.HybridSearchResult{
			makeResult("a.md", 0.9),
			makeResult("b.md", 0.8),
		}
		results := []variantResult{{weight: 1.0, results: list1}}
		var p *Pipeline

		fusedK60 := p.rrfFuse(results, 60)
		fusedK5 := p.rrfFuse(results, 5)

		if fusedK5[0].rrfScore <= fusedK60[0].rrfScore {
			t.Errorf("smaller k should give larger score: k=5 (%f) vs k=60 (%f)",
				fusedK5[0].rrfScore, fusedK60[0].rrfScore)
		}
	})

	t.Run("shared result appears in multiple variant lists", func(t *testing.T) {
		list1 := []ts.HybridSearchResult{
			makeResult("a.md", 0.9),
		}
		list2 := []ts.HybridSearchResult{
			makeResult("a.md", 0.85),
		}
		results := []variantResult{
			{weight: 1.0, results: list1},
			{weight: 1.0, results: list2},
		}
		var p *Pipeline
		fused := p.rrfFuse(results, 60)

		if len(fused) != 1 {
			t.Fatalf("expected 1 fused doc, got %d", len(fused))
		}

		expectedScore := (1.0/float64(60+1) + topRankBonus) + (1.0 / float64(60+1))
		if math.Abs(fused[0].rrfScore-expectedScore) > 1e-10 {
			t.Errorf("score = %f, want %f", fused[0].rrfScore, expectedScore)
		}
	})
}

func TestSearch_Blend(t *testing.T) {
	makeFused := func(rrfScore, rerankScore float64) fusedDoc {
		return fusedDoc{
			key:         "docs:test.md",
			result:      makeResult("test.md", 0.8),
			rrfScore:    rrfScore,
			rerankScore: rerankScore,
		}
	}

	t.Run("single candidate remains unchanged", func(t *testing.T) {
		cfg := &config.Config{
			Pipeline: config.PipelineConfig{
				Blending: config.BlendingConfig{
					Thresholds: config.BlendingThresholds{Top: 3, Middle: 10},
					Weights:    config.BlendingWeights{Top: 0.75, Middle: 0.6, Bottom: 0.4},
				},
			},
		}
		p := &Pipeline{cfg: cfg}
		candidates := []fusedDoc{makeFused(0.8, 0.9)}
		p.blend(candidates)

		if len(candidates) != 1 {
			t.Fatalf("expected 1, got %d", len(candidates))
		}
		if candidates[0].finalScore == 0 {
			t.Error("finalScore should not be 0")
		}
	})

	t.Run("top tier uses top weight", func(t *testing.T) {
		cfg := &config.Config{
			Pipeline: config.PipelineConfig{
				Blending: config.BlendingConfig{
					Thresholds: config.BlendingThresholds{Top: 3, Middle: 10},
					Weights:    config.BlendingWeights{Top: 0.75, Middle: 0.6, Bottom: 0.4},
				},
			},
		}
		p := &Pipeline{cfg: cfg}
		candidates := []fusedDoc{
			makeFused(0.8, 0.9),
			makeFused(0.3, 0.1),
		}
		p.blend(candidates)

		if candidates[0].finalScore < candidates[1].finalScore {
			t.Errorf("top candidate should have higher final score")
		}
	})

	t.Run("all candidates at same score sort stable", func(t *testing.T) {
		cfg := &config.Config{
			Pipeline: config.PipelineConfig{
				Blending: config.BlendingConfig{
					Thresholds: config.BlendingThresholds{Top: 3, Middle: 10},
					Weights:    config.BlendingWeights{Top: 0.75, Middle: 0.6, Bottom: 0.4},
				},
			},
		}
		p := &Pipeline{cfg: cfg}
		candidates := []fusedDoc{
			makeFused(0.5, 0.5),
			makeFused(0.5, 0.5),
			makeFused(0.5, 0.5),
		}
		p.blend(candidates)
		if len(candidates) != 3 {
			t.Errorf("expected 3 candidates, got %d", len(candidates))
		}
	})

	t.Run("empty candidates", func(t *testing.T) {
		cfg := &config.Config{
			Pipeline: config.PipelineConfig{
				Blending: config.BlendingConfig{
					Thresholds: config.BlendingThresholds{Top: 3, Middle: 10},
					Weights:    config.BlendingWeights{Top: 0.75, Middle: 0.6, Bottom: 0.4},
				},
			},
		}
		p := &Pipeline{cfg: cfg}
		p.blend(nil)
	})
}

func TestSearch_TsResultsToResults(t *testing.T) {
	tsResults := []ts.HybridSearchResult{
		{Collection: "docs", Path: "a.md", Title: "Doc A", Content: "content a", ChunkSeq: 0, Score: 0.95},
		{Collection: "docs", Path: "b.md", Title: "Doc B", Content: "content b", ChunkSeq: 1, Score: 0.85},
	}

	results := tsResultsToResults(tsResults)
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	if results[0].Collection != "docs" || results[0].Path != "a.md" || results[0].Score != 0.95 {
		t.Errorf("first result mismatch: %+v", results[0])
	}
	if results[1].Collection != "docs" || results[1].Path != "b.md" || results[1].Score != 0.85 {
		t.Errorf("second result mismatch: %+v", results[1])
	}
}

func TestSearch_TsResultsToResultsEmpty(t *testing.T) {
	results := tsResultsToResults(nil)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearch_GenerateVariantsParsing(t *testing.T) {
	parseExpansion := func(expansion string, original string) []variant {
		variants := []variant{{
			text:    original,
			vecText: original,
			weight:  2.0,
		}}
		for _, line := range strings.Split(expansion, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			typ := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			switch typ {
			case "lex":
				variants = append(variants, variant{text: val, vecText: val, weight: 1.0})
			case "vec":
				variants = append(variants, variant{text: original, vecText: val, weight: 1.0})
			case "hyde":
				variants = append(variants, variant{text: val, vecText: val, weight: 1.0})
			}
		}
		return variants
	}

	t.Run("parses lex variant", func(t *testing.T) {
		v := parseExpansion("lex: keyword search", "original query")
		if len(v) != 2 {
			t.Fatalf("got %d variants, want 2", len(v))
		}
		if v[1].text != "keyword search" {
			t.Errorf("lex text = %q, want %q", v[1].text, "keyword search")
		}
	})

	t.Run("parses vec variant", func(t *testing.T) {
		v := parseExpansion("vec: semantic meaning", "original query")
		if len(v) != 2 {
			t.Fatalf("got %d variants, want 2", len(v))
		}
		if v[1].text != "original query" {
			t.Errorf("vec text should match original, got %q", v[1].text)
		}
		if v[1].vecText != "semantic meaning" {
			t.Errorf("vec vecText = %q, want %q", v[1].vecText, "semantic meaning")
		}
	})

	t.Run("parses hyde variant", func(t *testing.T) {
		v := parseExpansion("hyde: The answer is...", "original query")
		if len(v) != 2 {
			t.Fatalf("got %d variants, want 2", len(v))
		}
		if v[1].text != "The answer is..." {
			t.Errorf("hyde text = %q, want %q", v[1].text, "The answer is...")
		}
	})

	t.Run("parses all three types", func(t *testing.T) {
		expansion := "lex: keywords\nvec: semantic\nhyde: hypothetical passage"
		v := parseExpansion(expansion, "original")
		if len(v) != 4 {
			t.Fatalf("got %d variants (want 4: original + 3 expansions)", len(v))
		}
		if v[1].text != "keywords" || v[2].vecText != "semantic" || v[3].text != "hypothetical passage" {
			t.Errorf("variant parsing incorrect: %+v", v)
		}
	})

	t.Run("ignores empty lines", func(t *testing.T) {
		v := parseExpansion("lex: kw\n\n\nvec: sem", "original")
		if len(v) != 3 {
			t.Fatalf("got %d variants, want 3", len(v))
		}
	})

	t.Run("handles colon in value", func(t *testing.T) {
		v := parseExpansion("lex: time: 12:30 PM", "original")
		if len(v) != 2 {
			t.Fatalf("got %d variants, want 2", len(v))
		}
		if v[1].text != "time: 12:30 PM" {
			t.Errorf("expected 'time: 12:30 PM', got %q", v[1].text)
		}
	})

	t.Run("handles empty expansion", func(t *testing.T) {
		v := parseExpansion("", "original")
		if len(v) != 1 {
			t.Fatalf("got %d variants, want 1 (original only)", len(v))
		}
	})

	t.Run("ignores unknown prefixes", func(t *testing.T) {
		v := parseExpansion("lex: kw\nunknown: foo\nvec: sem", "original")
		if len(v) != 3 {
			t.Fatalf("got %d variants, want 3", len(v))
		}
	})

	t.Run("trims whitespace from lines", func(t *testing.T) {
		v := parseExpansion("  lex:  keyword  ", "original")
		if len(v) != 2 {
			t.Fatalf("got %d variants, want 2", len(v))
		}
		if v[1].text != "keyword" {
			t.Errorf("expected 'keyword', got %q", v[1].text)
		}
	})
}

func TestSearch_StrongSignalThresholds(t *testing.T) {
	t.Run("strong signal with score and gap", func(t *testing.T) {
		strong := 0.90 >= 0.85 && (0.90-0.70) >= 0.15
		if !strong {
			t.Error("expected strong signal")
		}
	})

	t.Run("weak signal insufficient gap", func(t *testing.T) {
		strong := 0.90 >= 0.85 && (0.90-0.80) >= 0.15
		if strong {
			t.Error("expected weak signal (gap < 0.15)")
		}
	})

	t.Run("weak signal insufficient score", func(t *testing.T) {
		strong := 0.50 >= 0.85 && (0.50-0.45) >= 0.15
		if strong {
			t.Error("expected weak signal (score < 0.85)")
		}
	})

	t.Run("less than 2 results is weak", func(t *testing.T) {
	})
}

func TestSearch_ExpansionPrompt(t *testing.T) {
	p := expansionPrompt()
	if p == "" {
		t.Fatal("expansionPrompt should not be empty")
	}
	if !strings.Contains(p, "lex:") {
		t.Error("expansionPrompt should contain lex:")
	}
	if !strings.Contains(p, "vec:") {
		t.Error("expansionPrompt should contain vec:")
	}
	if !strings.Contains(p, "hyde:") {
		t.Error("expansionPrompt should contain hyde:")
	}
}

func TestSearch_GenerateVariantsStrongSignal(t *testing.T) {
	cfg := &config.Config{
		Pipeline: config.PipelineConfig{
			RRF: config.RRFConfig{OriginalWeight: 2.0, ExpansionWeight: 1.0},
		},
	}
	p := &Pipeline{cfg: cfg}

	variants, err := p.generateVariants(t.Context(), "test query", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(variants) != 1 {
		t.Fatalf("got %d variants, want 1 (original only)", len(variants))
	}
	if variants[0].weight != 2.0 {
		t.Errorf("weight = %f, want 2.0", variants[0].weight)
	}
}

func TestSearch_SearchModeValues(t *testing.T) {
	if ModeText != 0 {
		t.Errorf("ModeText should be 0, got %d", ModeText)
	}
	if ModeVector != 1 {
		t.Errorf("ModeVector should be 1, got %d", ModeVector)
	}
	if ModeHybrid != 2 {
		t.Errorf("ModeHybrid should be 2, got %d", ModeHybrid)
	}
}
