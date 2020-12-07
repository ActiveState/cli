package colorize

type Entry struct {
	Line   string
	Length int
}

func GetCroppedText(text string, maxLen int) []Entry {
	entries := make([]Entry, 0)
	entryText := ""
	currentPosition := 0
	runesWritten := 0
	matches := colorRx.FindAllSubmatchIndex([]byte(text), -1)
	runeText := []rune(text)
	last := len(runeText) - 1

	for currentPosition < len(runeText) {
		// If we reach an index that we recognize (ie. the start of a tag)
		// then we write the whole tag, otherwise write by rune
		for _, match := range matches {
			start, stop := match[0], match[1]
			if currentPosition == start {
				entryText += string(runeText[currentPosition:stop])
				currentPosition = stop
			}
		}

		if currentPosition > last {
			entries = append(entries, Entry{entryText, runesWritten})
			break
		}

		// Reached end of line
		if runesWritten == maxLen {
			entries = append(entries, Entry{entryText, runesWritten})
			runesWritten = 0
			entryText = ""
			continue
		}

		// Text already has line ending, so terminate the line here
		if runeText[currentPosition] == '\n' {
			currentPosition++
			entries = append(entries, Entry{entryText, runesWritten})
			runesWritten = 0
			entryText = ""
			continue
		}

		if currentPosition == last {
			entryText += string(runeText[currentPosition])
			entries = append(entries, Entry{entryText, runesWritten + 1})
		}

		entryText += string(runeText[currentPosition])
		currentPosition++
		runesWritten++
	}

	return entries
}
