package colorize

type CroppedLines []CroppedLine

type CroppedLine struct {
	Line   string
	Length int
}

func GetCroppedText(text string, maxLen int) CroppedLines {
	entries := make([]CroppedLine, 0)
	colorCodes := colorRx.FindAllStringSubmatchIndex(text, -1)

	entry := CroppedLine{}
	for pos, amend := range text {
		inColorTag := inRange(pos, colorCodes)
		lineEnd := amend == '\n'

		if !lineEnd {
			entry.Line += string(amend)
			if !inColorTag {
				entry.Length++
			}
		}

		// Ensure the next position is not within a color tag and check conditions that would end this entry
		if lineEnd || (!inRange(pos+1, colorCodes) && (entry.Length == maxLen || lineEnd || pos == len(text)-1)) {
			entries = append(entries, entry)
			entry = CroppedLine{}
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
