package logging

// Ring buffer structure, but without a start position for reading. All reads are tail reads with
// no concept for what has already been read. The use case here is for logging to keep track of the
// last X bytes written.
type ringBuffer struct {
	buffer []byte
	size   int
	end    int
	full   bool
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{make([]byte, size), size, 0, false}
}

// Read reads the last len(p) bytes from the buffer.
// Typically len(p) should be the size of the buffer.
// This method satisfies the io.Reader interface.
func (b *ringBuffer) Read(p []byte) (int, error) {
	switch {
	case !b.full && b.end <= len(p): // entire buffer fits in p
		copy(p, b.buffer[:b.end]) // fill p with buffer
		return b.end, nil
	case !b.full && b.end > len(p): // entire buffer does not fit in p (not wrapped)
		copy(p, b.buffer[b.end-len(p):b.end]) // fill p with trailing buffer
		return len(p), nil
	case b.size <= len(p): // entire buffer fits in p (wrapped)
		copy(p, b.buffer[b.end:])        // fill p with beginning of buffer
		copy(p[b.size-b.end:], b.buffer) // finish filling with wrapped remainder
		return b.size, nil
	default: // b.size > len(p): // entire buffer does not fit in p (wrapped)
		// Fill p with trailing buffer, accounting for wrapping.
		for i, read := (b.end-len(p)+b.size)%b.size, 0; read < len(p); read, i = read+1, (i+1)%b.size {
			p[read] = b.buffer[i]
		}
		return len(p), nil
	}
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
