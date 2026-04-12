package stringutils

import (
	"math"
	"strings"
	"unicode"
)

// Replacer is a strategy for finding/replacing text in files
type Replacer interface {
	// Find yields candidate match strings from content
	Find(content, search string) <-chan string
}

// SimpleReplacer does exact string matching
type SimpleReplacer struct{}

func (r *SimpleReplacer) Find(content, search string) <-chan string {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)
		if idx := strings.Index(content, search); idx >= 0 {
			ch <- content[idx : idx+len(search)]
		}
	}()
	return ch
}

// LineTrimmedReplacer matches with trimmed whitespace per line
type LineTrimmedReplacer struct{}

func (r *LineTrimmedReplacer) Find(content, search string) <-chan string {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)

		searchLines := strings.Split(search, "\n")
		contentLines := strings.Split(content, "\n")

		// Trim each line for comparison
		trimmedSearch := make([]string, len(searchLines))
		for i, l := range searchLines {
			trimmedSearch[i] = strings.TrimSpace(l)
		}

		// Slide through content lines
		for i := 0; i <= len(contentLines)-len(searchLines); i++ {
			match := true
			for j := 0; j < len(searchLines); j++ {
				if strings.TrimSpace(contentLines[i+j]) != trimmedSearch[j] {
					match = false
					break
				}
			}
			if match {
				// Reconstruct original content block
				start := lineStart(content, i)
				end := lineEnd(content, i+len(searchLines)-1)
				ch <- content[start:end]
				return
			}
		}
	}()
	return ch
}

// BlockAnchorReplacer matches first/last lines exactly, uses Levenshtein for middle
type BlockAnchorReplacer struct{}

func (r *BlockAnchorReplacer) Find(content, search string) <-chan string {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)

		searchLines := strings.Split(search, "\n")
		if len(searchLines) < 3 {
			return // Need at least 3 lines for anchor matching
		}

		contentLines := strings.Split(content, "\n")
		firstSearch := searchLines[0]
		lastSearch := searchLines[len(searchLines)-1]

		// Find all positions where first and last lines match
		type candidate struct {
			start int
			end   int
		}
		var candidates []candidate

		for i := 0; i <= len(contentLines)-len(searchLines); i++ {
			if contentLines[i] != firstSearch {
				continue
			}
			if contentLines[i+len(searchLines)-1] != lastSearch {
				continue
			}

			// Calculate Levenshtein distance for middle lines
			totalDist := 0
			for j := 1; j < len(searchLines)-1; j++ {
				dist := levenshtein(contentLines[i+j], searchLines[j])
				totalDist += dist
			}

			avgDist := float64(totalDist) / float64(len(searchLines)-2)
			maxLen := 0
			for _, l := range searchLines[1 : len(searchLines)-1] {
				if len(l) > maxLen {
					maxLen = len(l)
				}
			}

			// Threshold: 0.0 for single candidate, 0.3 for multiple
			threshold := 0.3
			if avgDist/float64(maxLen+1) <= threshold {
				start := lineStart(content, i)
				end := lineEnd(content, i+len(searchLines)-1)
				candidates = append(candidates, candidate{start, end})
			}
		}

		if len(candidates) == 1 {
			ch <- content[candidates[0].start:candidates[0].end]
		}
	}()
	return ch
}

// WhitespaceNormalizedReplacer collapses all whitespace to single space
type WhitespaceNormalizedReplacer struct{}

func (r *WhitespaceNormalizedReplacer) Find(content, search string) <-chan string {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)

		normalize := func(s string) string {
			fields := strings.Fields(s)
			return strings.Join(fields, " ")
		}

		normSearch := normalize(search)
		normContent := normalize(content)

		if idx := strings.Index(normContent, normSearch); idx >= 0 {
			// Find corresponding position in original content
			origIdx := mapNormalizedPosition(content, idx, len(normSearch))
			if origIdx >= 0 {
				// Find end position
				searchLen := countOriginalChars(search)
				ch <- content[origIdx : origIdx+searchLen]
			}
		}
	}()
	return ch
}

// IndentationFlexibleReplacer strips minimum indentation from both search and content
type IndentationFlexibleReplacer struct{}

func (r *IndentationFlexibleReplacer) Find(content, search string) <-chan string {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)

		stripIndent := func(s string) (string, int) {
			lines := strings.Split(s, "\n")
			minIndent := math.MaxInt32
			for _, line := range lines {
				if strings.TrimSpace(line) == "" {
					continue
				}
				indent := 0
				for _, c := range line {
					if unicode.IsSpace(c) {
						indent++
					} else {
						break
					}
				}
				if indent < minIndent {
					minIndent = indent
				}
			}

			result := make([]string, len(lines))
			for i, line := range lines {
				if len(line) >= minIndent {
					result[i] = line[minIndent:]
				} else {
					result[i] = line
				}
			}
			return strings.Join(result, "\n"), minIndent
		}

		strippedSearch, _ := stripIndent(search)
		strippedContent, origIndent := stripIndent(content)

		if idx := strings.Index(strippedContent, strippedSearch); idx >= 0 {
			// Find original block
			searchLines := strings.Split(search, "\n")
			contentLines := strings.Split(content, "\n")

			for i := 0; i <= len(contentLines)-len(searchLines); i++ {
				match := true
				for j := 0; j < len(searchLines); j++ {
					stripI := func(s string) string {
						if len(s) >= origIndent {
							return s[origIndent:]
						}
						return s
					}
					if stripI(contentLines[i+j]) != stripI(searchLines[j]) {
						match = false
						break
					}
				}
				if match {
					start := lineStart(content, i)
					end := lineEnd(content, i+len(searchLines)-1)
					ch <- content[start:end]
					return
				}
			}
		}
	}()
	return ch
}

// EscapeNormalizedReplacer handles escaped strings (\n, \t, \\, etc.)
type EscapeNormalizedReplacer struct{}

func (r *EscapeNormalizedReplacer) Find(content, search string) <-chan string {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)

		unescape := func(s string) string {
			result := strings.ReplaceAll(s, "\\n", "\n")
			result = strings.ReplaceAll(result, "\\t", "\t")
			result = strings.ReplaceAll(result, "\\\\", "\\")
			result = strings.ReplaceAll(result, "\\r", "\r")
			return result
		}

		unescapedSearch := unescape(search)
		if idx := strings.Index(content, unescapedSearch); idx >= 0 {
			// Find the original escaped form
			origSearch := findEscapedForm(content, unescapedSearch)
			if origSearch != "" {
				ch <- origSearch
			}
		}
	}()
	return ch
}

// TrimmedBoundaryReplacer trims leading/trailing whitespace from search
type TrimmedBoundaryReplacer struct{}

func (r *TrimmedBoundaryReplacer) Find(content, search string) <-chan string {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)

		trimmed := strings.TrimSpace(search)
		if idx := strings.Index(content, trimmed); idx >= 0 {
			// Find the actual boundary in content
			start := idx
			for start > 0 && isWhitespace(content[start-1]) {
				start--
			}
			end := idx + len(trimmed)
			for end < len(content) && isWhitespace(content[end]) {
				end++
			}
			ch <- content[start:end]
		}
	}()
	return ch
}

// ContextAwareReplacer uses first/last lines as anchors, requires 50% middle line match
type ContextAwareReplacer struct{}

func (r *ContextAwareReplacer) Find(content, search string) <-chan string {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)

		searchLines := strings.Split(search, "\n")
		if len(searchLines) < 3 {
			return
		}

		contentLines := strings.Split(content, "\n")
		firstLine := searchLines[0]
		lastLine := searchLines[len(searchLines)-1]
		middleLines := searchLines[1 : len(searchLines)-1]

		for i := 0; i <= len(contentLines)-len(searchLines); i++ {
			if contentLines[i] != firstLine {
				continue
			}
			if contentLines[i+len(searchLines)-1] != lastLine {
				continue
			}

			// Check middle lines match percentage
			matched := 0
			for j := 0; j < len(middleLines); j++ {
				if contentLines[i+1+j] == middleLines[j] {
					matched++
				}
			}

			if float64(matched)/float64(len(middleLines)) >= 0.5 {
				start := lineStart(content, i)
				end := lineEnd(content, i+len(searchLines)-1)
				ch <- content[start:end]
				return
			}
		}
	}()
	return ch
}

// MultiOccurrenceReplacer yields all exact matches for replaceAll
type MultiOccurrenceReplacer struct{}

func (r *MultiOccurrenceReplacer) Find(content, search string) <-chan string {
	ch := make(chan string, len(content)/len(search)+1)
	go func() {
		defer close(ch)
		
		idx := 0
		for {
			pos := strings.Index(content[idx:], search)
			if pos < 0 {
				break
			}
			ch <- search
			idx += pos + len(search)
		}
	}()
	return ch
}

// Helper functions

func levenshtein(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			matrix[i][j] = min3(
				matrix[i-1][j]+1,
				matrix[i][j-1]+1,
				matrix[i-1][j-1]+cost,
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func min3(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

func lineStart(content string, lineNum int) int {
	if lineNum == 0 {
		return 0
	}
	pos := 0
	for i := 0; i < lineNum; i++ {
		idx := strings.Index(content[pos:], "\n")
		if idx < 0 {
			return pos
		}
		pos += idx + 1
	}
	return pos
}

func lineEnd(content string, lineNum int) int {
	pos := 0
	for i := 0; i <= lineNum; i++ {
		idx := strings.Index(content[pos:], "\n")
		if idx < 0 {
			return len(content)
		}
		pos += idx + 1
	}
	return pos
}

func mapNormalizedPosition(orig string, normIdx, normLen int) int {
	origPos := 0
	normPos := 0

	for normPos < normIdx && origPos < len(orig) {
		if isWhitespace(orig[origPos]) {
			// Skip all whitespace in original
			for origPos < len(orig) && isWhitespace(orig[origPos]) {
				origPos++
			}
			normPos++
		} else {
			origPos++
			normPos++
		}
	}

	return origPos
}

func countOriginalChars(s string) int {
	count := 0
	for _, r := range s {
		if !unicode.IsSpace(r) {
			count++
		}
	}
	return count
}

func findEscapedForm(content, unescaped string) string {
	// Try to find the escaped form that produced this unescaped string
	idx := strings.Index(content, unescaped)
	if idx >= 0 {
		return unescaped
	}
	return ""
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
