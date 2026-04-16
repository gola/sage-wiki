package extract

import (
	"regexp"
	"strings"
)

// Section represents a heading-bounded section of a document.
type Section struct {
	Heading     string
	Content     string
	StartOffset int
	EndOffset   int
	Level       int // heading level (1-6)
}

var headingRe = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)

// SplitByHeadings splits a document into sections at markdown heading boundaries.
// Returns the full document as a single section if no headings are found
// or if the document is below the threshold (in characters).
// A threshold of 0 disables the threshold check (always splits).
func SplitByHeadings(content string, threshold int) []Section {
	if threshold > 0 && len([]rune(content)) < threshold {
		return []Section{{
			Content:   content,
			EndOffset: len(content),
		}}
	}

	matches := headingRe.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return []Section{{
			Content:   content,
			EndOffset: len(content),
		}}
	}

	var sections []Section

	// Content before first heading (preamble)
	if matches[0][0] > 0 {
		preamble := strings.TrimSpace(content[:matches[0][0]])
		if preamble != "" {
			sections = append(sections, Section{
				Content:     preamble,
				StartOffset: 0,
				EndOffset:   matches[0][0],
			})
		}
	}

	for i, match := range matches {
		// match[0]:match[1] = full heading line
		// match[2]:match[3] = hashes (e.g., "##")
		// match[4]:match[5] = heading text
		level := match[3] - match[2] // number of # characters
		heading := content[match[4]:match[5]]

		// Section content = from heading start to next heading start (or end of doc)
		sectionStart := match[0]
		var sectionEnd int
		if i+1 < len(matches) {
			sectionEnd = matches[i+1][0]
		} else {
			sectionEnd = len(content)
		}

		sectionContent := strings.TrimSpace(content[sectionStart:sectionEnd])

		sections = append(sections, Section{
			Heading:     heading,
			Content:     sectionContent,
			StartOffset: sectionStart,
			EndOffset:   sectionEnd,
			Level:       level,
		})
	}

	return sections
}

// SectionsContaining returns sections whose heading or content contains
// any of the given terms (case-insensitive). Used to select relevant
// sections for a concept during article writing.
func SectionsContaining(sections []Section, terms []string) []Section {
	var matches []Section
	for _, s := range sections {
		lower := strings.ToLower(s.Heading + " " + s.Content)
		for _, term := range terms {
			if strings.Contains(lower, strings.ToLower(term)) {
				matches = append(matches, s)
				break
			}
		}
	}
	return matches
}
