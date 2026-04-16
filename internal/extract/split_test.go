package extract

import (
	"strings"
	"testing"
)

func TestSplitByHeadings_Basic(t *testing.T) {
	content := `# Introduction

This is the intro.

## Methods

Some methods here.

## Results

The results section.

### Sub-results

Details.
`
	sections := SplitByHeadings(content, 0)
	if len(sections) != 4 {
		t.Fatalf("expected 4 sections, got %d", len(sections))
	}

	if sections[0].Heading != "Introduction" {
		t.Errorf("section 0 heading = %q, want Introduction", sections[0].Heading)
	}
	if sections[0].Level != 1 {
		t.Errorf("section 0 level = %d, want 1", sections[0].Level)
	}

	if sections[1].Heading != "Methods" {
		t.Errorf("section 1 heading = %q, want Methods", sections[1].Heading)
	}
	if sections[1].Level != 2 {
		t.Errorf("section 1 level = %d, want 2", sections[1].Level)
	}

	if sections[2].Heading != "Results" {
		t.Errorf("section 2 heading = %q, want Results", sections[2].Heading)
	}

	if sections[3].Heading != "Sub-results" {
		t.Errorf("section 3 heading = %q, want Sub-results", sections[3].Heading)
	}
	if sections[3].Level != 3 {
		t.Errorf("section 3 level = %d, want 3", sections[3].Level)
	}
}

func TestSplitByHeadings_BelowThreshold(t *testing.T) {
	content := "# Short\n\nSmall doc."
	sections := SplitByHeadings(content, 15000)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section (below threshold), got %d", len(sections))
	}
	if sections[0].Content != content {
		t.Error("below-threshold section should contain full document")
	}
}

func TestSplitByHeadings_NoHeadings(t *testing.T) {
	content := "Just a plain text document without any headings.\nMultiple lines of content."
	sections := SplitByHeadings(content, 0)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section (no headings), got %d", len(sections))
	}
	if sections[0].Content != content {
		t.Error("no-heading section should contain full document")
	}
}

func TestSplitByHeadings_Preamble(t *testing.T) {
	content := `Some text before any heading.

# First Heading

Content under first heading.
`
	sections := SplitByHeadings(content, 0)
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections (preamble + heading), got %d", len(sections))
	}
	if sections[0].Heading != "" {
		t.Errorf("preamble should have empty heading, got %q", sections[0].Heading)
	}
	if !strings.Contains(sections[0].Content, "Some text before") {
		t.Error("preamble should contain pre-heading text")
	}
	if sections[1].Heading != "First Heading" {
		t.Errorf("second section heading = %q, want First Heading", sections[1].Heading)
	}
}

func TestSplitByHeadings_LargeDoc(t *testing.T) {
	// Build a 20K char doc with 4 headings
	var b strings.Builder
	for i := 0; i < 4; i++ {
		b.WriteString("## Section " + string(rune('A'+i)) + "\n\n")
		b.WriteString(strings.Repeat("Content for this section. ", 200))
		b.WriteString("\n\n")
	}
	content := b.String()

	sections := SplitByHeadings(content, 15000)
	if len(sections) != 4 {
		t.Fatalf("expected 4 sections, got %d", len(sections))
	}
	for i, s := range sections {
		expected := "Section " + string(rune('A'+i))
		if s.Heading != expected {
			t.Errorf("section %d heading = %q, want %q", i, s.Heading, expected)
		}
	}
}

func TestSectionsContaining(t *testing.T) {
	sections := []Section{
		{Heading: "Introduction", Content: "Overview of attention mechanisms."},
		{Heading: "Methods", Content: "We use flash attention for speed."},
		{Heading: "Results", Content: "The transformer performed well."},
	}

	matches := SectionsContaining(sections, []string{"attention"})
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches for 'attention', got %d", len(matches))
	}

	matches = SectionsContaining(sections, []string{"transformer"})
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for 'transformer', got %d", len(matches))
	}

	matches = SectionsContaining(sections, []string{"nonexistent"})
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches for 'nonexistent', got %d", len(matches))
	}
}
