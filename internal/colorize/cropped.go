package colorize

type Entry struct {
	Line   string
	Length int
}

func GetCroppedText(text string, maxLen int) []Entry {
	var line string // the line we're building, including the color codes
	var plainTextPosition int
	var plainTextWritten int

	entries := make([]Entry, 0)
	colorCodes := colorRx.FindAllStringSubmatchIndex(text, -1)
	runeText := []rune(text)

	for plainTextPosition < len(runeText) {
		// If we reach an index that we recognize (ie. the start of a tag)
		// then we write the whole tag, otherwise write by rune
		for _, match := range colorCodes {
			start, stop := match[0], match[1]
			if plainTextPosition == start {
				line += string(runeText[plainTextPosition:stop])
				plainTextPosition = stop
			}
		}

		if plainTextPosition > len(runeText)-1 {
			entries = append(entries, Entry{line, plainTextWritten})
			break
		}

		// Reached end of line
		if plainTextWritten == maxLen {
			entries = append(entries, Entry{line, plainTextWritten})
			plainTextWritten = 0
			line = ""
			continue
		}

		// Text already has line ending, so terminate the line here
		if runeText[plainTextPosition] == '\n' {
			plainTextPosition++
			entries = append(entries, Entry{line, plainTextWritten})
			plainTextWritten = 0
			line = ""
			continue
		}

		if plainTextPosition == len(runeText)-1 {
			line += string(runeText[plainTextPosition])
			entries = append(entries, Entry{line, plainTextWritten + 1})
		}

		line += string(runeText[plainTextPosition])
		plainTextPosition++
		plainTextWritten++
	}

	return entries
}
