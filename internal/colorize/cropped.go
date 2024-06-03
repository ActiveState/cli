package colorize

import (
	"regexp"
)

type CroppedLines []CroppedLine

type CroppedLine struct {
	Line   string
	Length int
}

func (c CroppedLines) String() string {
	var result string
	for _, crop := range c {
		result = result + crop.Line
	}

	return result
}

var indentRegexp = regexp.MustCompile(`^([ ]+)`)

func GetCroppedText(text string, maxLen int, includeLineEnds bool) CroppedLines {
	indent := ""
	if indentMatch := indentRegexp.FindStringSubmatch(text); indentMatch != nil {
		indent = indentMatch[0]
		maxLen -= len(indent)
	}

	entries := make([]CroppedLine, 0)
	colorCodes := colorRx.FindAllStringSubmatchIndex(text, -1)

	isLineEnd := false
	entry := CroppedLine{}
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
			wrappedLength := 0
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
				if i > 0 && i < len(entry.Line)-1 {
					wrapped = indent + entry.Line[i:]
					entry.Line = entry.Line[:i]
					entry.Length -= wrappedLength
					isLineEnd = true // emulate for wrapping purposes
				} else {
					wrappedLength = 0 // reset
				}
			}
			entries = append(entries, entry)
			entry = CroppedLine{Line: wrapped, Length: wrappedLength}
		}

		if isLineEnd && includeLineEnds {
			entries = append(entries, CroppedLine{"\n", 1})
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

func isSpace(b byte) bool { return b == ' ' || b == '\t' }

func isUTF8TrailingByte(b byte) bool {
	return b >= 0x80 && b < 0xC0
}
