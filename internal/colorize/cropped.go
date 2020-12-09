package colorize

type CroppedLines []CroppedLine

type CroppedLine struct {
	Line   string
	Length int
}

func GetCroppedText(text []rune, maxLen int) CroppedLines {
	entries := make([]CroppedLine, 0)
	colorCodes := colorRx.FindAllStringSubmatchIndex(string(text), -1)

	entry := CroppedLine{}
	for pos := 0; pos < len(text); pos++ {
		inColorTag := inRange(pos, colorCodes)
		amend := text[pos]
		lineEnd := amend == '\n'

		if !lineEnd {
			entry.Line += string(amend)
			if !inColorTag {
				entry.Length++
			}
		}

		if !inRange(pos+1, colorCodes) && (entry.Length == maxLen || lineEnd || pos == len(text)-1) {
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
