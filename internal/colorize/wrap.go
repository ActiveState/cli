package colorize

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/logging"
)

type WrappedLines []WrappedLine

type WrappedLine struct {
	Line   string
	Length int
}

func (c WrappedLines) String() string {
	var result string
	for _, crop := range c {
		result = result + crop.Line
	}

	return result
}

var indentRegexp = regexp.MustCompile(`^([ ]+)`)
var isLinkRegexp = regexp.MustCompile(`\s*(\[[^\]]+\])?https?://`)

func Wrap(text string, maxLen int, includeLineEnds bool, continuation string) WrappedLines {
	indent := ""
	if indentMatch := indentRegexp.FindStringSubmatch(text); indentMatch != nil {
		indent = indentMatch[0]
		if len(text) > len(indent) && strings.HasPrefix(text[len(indent):], "• ") {
			indent += "  "
		}
	}

	maxLen -= len(continuation)

	entries := make([]WrappedLine, 0)
	colorCodes := colorRx.FindAllStringSubmatchIndex(text, -1)
	colorNames := colorRx.FindAllStringSubmatch(text, -1)

	isLineEnd := false
	entry := WrappedLine{}
	for pos, amend := range text {
		inColorTag := inRange(pos, colorCodes)

		isLineEnd = amend == '\n'

		if !isLineEnd {
			entry.Line += string(amend)
			if !inColorTag {
				entry.Length++
			}
		}

		// Ensure the next position is not within a color tag and check conditions that would end this entry
		if isLineEnd || (!inRange(pos+1, colorCodes) && (entry.Length == maxLen || pos == len(text)-1)) {
			wrapped := ""
			wrappedLength := len(indent)
			nextCharIsSpace := pos+1 < len(text) && isSpace(text[pos+1])
			if !isLineEnd && entry.Length == maxLen && !nextCharIsSpace && pos < len(text)-1 {
				// Put the current word on the next line, if possible.
				// Find the start of the current word and its printed length, taking color ranges and
				// multi-byte characters into account.
				i := len(entry.Line) - 1
				for ; i > 0; i-- {
					if isSpace(entry.Line[i]) {
						i++ // preserve trailing space
						break
					}
					if !inRange(pos-(len(entry.Line)-i), colorCodes) && !isUTF8TrailingByte(entry.Line[i]) {
						wrappedLength++
					}
				}
				// Extract the word from the current line if it doesn't start the line.
				if i > 0 && i < len(entry.Line)-1 && !isLinkRegexp.MatchString(entry.Line[i:]) {
					tag := colorTag(pos, colorCodes, colorNames)
					if continuation != "" && tag != "" {
						// Do not colorize the continuation.
						wrapped = fmt.Sprintf("%s[/RESET]%s%s%s", indent, continuation, tag, entry.Line[i:])
					} else {
						wrapped = indent + continuation + entry.Line[i:]
					}
					logging.Debug("continuation: '%s'", continuation)
					logging.Debug("wrapped: '%s'", wrapped)
					entry.Line = entry.Line[:i]
					entry.Length -= wrappedLength
					isLineEnd = true // emulate for wrapping purposes
				} else {
					wrappedLength = len(indent) // reset
				}
			}
			entries = append(entries, entry)
			entry = WrappedLine{Line: wrapped, Length: wrappedLength}
		}

		if isLineEnd && includeLineEnds {
			entries = append(entries, WrappedLine{"\n", 1})
		}
	}

	return entries
}

func inRange(pos int, ranges [][]int) bool {
	for _, intRange := range ranges {
		start, stop := intRange[0], intRange[1]
		if pos >= start && pos <= stop-1 {
			return true
		}
	}
	return false
}

// colorTag returns the currently active color tag (if any) at the given position.
func colorTag(pos int, ranges [][]int, names [][]string) string {
	for i, intRange := range ranges {
		if pos < intRange[0] {
			continue // before [COLOR]
		}
		if i < len(ranges)-1 || pos < ranges[i+1][0] {
			return names[i][0] // missing [/RESET] or between [COLOR] and [/RESET]
		}
	}
	return ""
}

func isSpace(b byte) bool { return b == ' ' || b == '\t' }

func isUTF8TrailingByte(b byte) bool {
	return b >= 0x80 && b < 0xC0
}
