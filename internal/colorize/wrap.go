package colorize

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

type WrappedLines []WrappedLine

type WrappedLine struct {
	Line   string
	Length int
}

func (c WrappedLines) String() string {
	var result string
	for _, wrapped := range c {
		result = result + wrapped.Line
	}

	return result
}

var indentRegexp = regexp.MustCompile(`^([ ]+)`)
var isLinkRegexp = regexp.MustCompile(`\s*(\[[^\]]+\])?https?://`)

func Wrap(text string, maxLen int, includeLineEnds bool, continuation string) WrappedLines {
	// Determine indentation of wrapped lines based on any leading indentation.
	indent := ""
	if indentMatch := indentRegexp.FindStringSubmatch(text); indentMatch != nil {
		indent = indentMatch[0]
		if len(text) > len(indent) && strings.HasPrefix(text[len(indent):], "• ") {
			// If text to wrap is a bullet item, indent extra to be flush after bullet.
			indent += "  "
		}
	}

	// If wrapping includes continuation text, reduce maximum wrapping length accordingly.
	maxLen -= utf8.RuneCountInString(StripColorCodes(continuation))

	entries := make([]WrappedLine, 0)
	colorCodes := colorRx.FindAllStringSubmatchIndex(text, -1)
	colorNames := colorRx.FindAllStringSubmatch(text, -1)

	entry := WrappedLine{}
	// Iterate over the text, one character at a time, and construct wrapped lines while doing so.
	for pos, amend := range text {
		isLineEnd := amend == '\n'
		isTextEnd := pos == len(text)-1

		// Add the current character to the wrapped line.
		// Update the wrapped line's length as long as the added character is not part of a tag like
		// [ERROR] or [/RESET].
		if !isLineEnd {
			entry.Line += string(amend)
			if !inRange(pos, colorCodes) {
				entry.Length++
			}
		}
		atWrapPosition := entry.Length == maxLen

		// When we've reached the end of the line, either naturally (line or text end), or when we've
		// reached the wrap position (maximum length), we need to wrap the current word (if any) and
		// set up the next (wrapped) line.
		// Note that if we've reached the wrap position but there's a tag immediately after it, we want
		// to include the tag, so do not wrap in that case.
		if isLineEnd || isTextEnd || (atWrapPosition && !inRange(pos+1, colorCodes)) {
			wrapped := ""                // the start of the next (wrapped) line
			wrappedLength := len(indent) // the current length of the next (wrapped) line

			// We need to prepare the next (wrapped) line unless we're at line end, text end, not at wrap
			// position, or the next character is a space (i.e. no wrapping needed).
			nextCharIsSpace := !isTextEnd && isSpace(text[pos+1])
			if !isLineEnd && !isTextEnd && atWrapPosition && !nextCharIsSpace {
				// Determine the start of the current word along with its printed length (taking color
				// ranges and multi-byte characters into account).
				// We need to know these things in order to put it on the next line (if possible).
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

				// We can wrap this word on the next line as long as it's not at the beginning of the
				// current line and it's not part of a hyperlink.
				canWrap := i > 0 && !isLinkRegexp.MatchString(entry.Line[i:])
				if canWrap {
					wrapped = entry.Line[i:]
					entry.Line = entry.Line[:i]
					entry.Length -= wrappedLength
					isLineEnd = true // emulate for wrapping purposes

					// Prepend the continuation string to the wrapped line, and indent it as necessary.
					// The continuation itself should not be tagged with anything like [ERROR], but any text
					// after the continuation should be tagged.
					if continuation != "" {
						if tags := colorTags(pos, colorCodes, colorNames); len(tags) > 0 {
							wrapped = fmt.Sprintf("%s[/RESET]%s%s%s", indent, continuation, strings.Join(tags, ""), wrapped)
						} else {
							wrapped = indent + continuation + wrapped
						}
					} else {
						wrapped = indent + wrapped
					}
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

// colorTags returns the currently active color tags (if any) at the given position.
func colorTags(pos int, ranges [][]int, names [][]string) []string {
	tags := make([]string, 0)
	for i, intRange := range ranges {
		if pos < intRange[0] {
			break // before [COLOR]
		}
		if pos > intRange[0] {
			if names[i][1] == "/RESET" {
				tags = make([]string, 0) // clear
			} else {
				tags = append(tags, names[i][0])
			}
		}
	}
	return tags
}

func isSpace(b byte) bool { return b == ' ' || b == '\t' }

func isUTF8TrailingByte(b byte) bool {
	return b >= 0x80 && b < 0xC0
}
