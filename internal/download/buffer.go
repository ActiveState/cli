package download

import (
	"github.com/aws/aws-sdk-go/aws"
)

type WriteAtBuffer struct {
	*aws.WriteAtBuffer
	cb func(int)
}

func NewWriteAtBuffer(buf []byte, cb func(int)) *WriteAtBuffer {
	return &WriteAtBuffer{aws.NewWriteAtBuffer(buf), cb}
}

func (b *WriteAtBuffer) WriteAt(p []byte, pos int64) (n int, err error) {
	pLen, err := b.WriteAtBuffer.WriteAt(p, pos)
	if pLen != 0 {
		b.cb(pLen)
	}
	return pLen, err
}
