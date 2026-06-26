package wheel

import (
	"bytes"
	"encoding/csv"
	"strconv"
)

// generator is recorded in WHEEL. It is intentionally version-free so the same
// source produces the same bytes across State Tool releases.
const generator = "state"

// record is one RECORD row: an archived file with its PEP 376 hash and size.
type record struct {
	name string
	hash string
	size int64
}

// buildMetadata returns the dist-info METADATA contents.
func buildMetadata(meta Metadata) []byte {
	var b bytes.Buffer
	b.WriteString("Metadata-Version: 2.1\n")
	b.WriteString("Name: " + meta.Name + "\n")
	b.WriteString("Version: " + meta.Version + "\n")
	if meta.Summary != "" {
		b.WriteString("Summary: " + meta.Summary + "\n")
	}
	return b.Bytes()
}

// buildWheelFile returns the dist-info WHEEL contents for a pure-Python wheel.
func buildWheelFile() []byte {
	var b bytes.Buffer
	b.WriteString("Wheel-Version: 1.0\n")
	b.WriteString("Generator: " + generator + "\n")
	b.WriteString("Root-Is-Purelib: true\n")
	b.WriteString("Tag: py3-none-any\n")
	return b.Bytes()
}

// buildRecord returns the dist-info RECORD contents: one CSV row per archived
// file, plus RECORD's own row with an empty hash and size.
func buildRecord(records []record, recordName string) []byte {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	for _, r := range records {
		_ = w.Write([]string{r.name, r.hash, strconv.FormatInt(r.size, 10)})
	}
	_ = w.Write([]string{recordName, "", ""})
	w.Flush()
	return b.Bytes()
}
