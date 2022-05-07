package flisten

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

type Dialer struct {
	net.Dialer
}

func (d *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, addr)
	if os.Getenv("FLISTEN_SLOW") != "" {
		conn = &slowConn{conn}
	}
	return conn, asConnRefusedError(asFileNotExistError(err))
}

type slowConn struct {
	net.Conn
}

func (c *slowConn) Read(b []byte) (n int, err error) {
	bs := make([]byte, 1)
	var i int
	for ; i < len(b); i++ {
		time.Sleep(time.Millisecond * 100)
		if _, err := c.Conn.Read(bs); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return i, fmt.Errorf("slow read: %w", err)
		}
		if os.Getenv("FLISTEN_DEBUG") != "" {
			fmt.Println(string(bs))
		}
		copy(b[i:i+1], bs)
	}
	time.Sleep(time.Second)
	return i, nil
}
