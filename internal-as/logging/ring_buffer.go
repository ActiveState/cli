package logging

// Ring buffer structure, but without a start position for reading. The use case here is for
// logging to keep track of the last X bytes written.
type ringBuffer struct {
	buffer []byte
	size   int
	end    int
	full   bool
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{make([]byte, size), size, 0, false}
}

// Read returns the contents of the buffer as a string.
func (b *ringBuffer) Read() string {
	if !b.full {
		return string(b.buffer[:b.end])
	}
	return string(b.buffer[b.end:]) + string(b.buffer[:b.end])
}

// Write writes the given bytes into the buffer, wrapping as necessary.
// This method satisfies the io.Writer interface.
func (b *ringBuffer) Write(p []byte) (int, error) {
	for _, c := range p {
		b.buffer[b.end] = c
		b.end = (b.end + 1) % b.size
		if !b.full && b.end == 0 {
			b.full = true
		}
	}
	return len(p), nil
}
