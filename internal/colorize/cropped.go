package colorize

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

func GetCroppedText(text string, maxLen int, includeLineEnds bool) CroppedLines {
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
			entries = append(entries, entry)
			entry = CroppedLine{}
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
