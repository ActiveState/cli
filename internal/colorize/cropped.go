package colorize

type Entry struct {
	Line   string
	Length int
}

func GetCroppedText(text string, maxLen int) []Entry {
	entries := make([]Entry, 0)
	entryText := ""
	pos := 0
	count := 0
	matches := colorRx.FindAllSubmatchIndex([]byte(text), -1)
	runeText := []rune(text)

	for pos < len(runeText) {
		// If we reach an index that we recognize (ie. the start of a tag)
		// then we write the whole tag, otherwise write by rune
		for _, match := range matches {
			start, stop := match[0], match[1]
			if pos == start {
				entryText += string(runeText[pos:stop])
				pos = stop
			}
		}

		if pos > len(runeText)-1 {
			entries = append(entries, Entry{entryText, count})
			break
		}

		if count == maxLen {
			entries = append(entries, Entry{entryText, count})
			count = 0
			entryText = ""
			continue
		}

		if runeText[pos] == '\n' {
			pos++
			entries = append(entries, Entry{entryText, count})
			count = 0
			entryText = ""
			continue
		}

		if pos == len(runeText)-1 {
			entryText += string(runeText[pos])
			entries = append(entries, Entry{entryText, count + 1})
		}

		entryText += string(runeText[pos])
		pos++
		count++
	}

	return entries
}
