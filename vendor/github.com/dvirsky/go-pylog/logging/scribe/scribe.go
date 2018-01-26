// This is a Scribe logging handler, that emits its messages to a scribe server
package scribe

import (
	"fmt"
	"io"

	"github.com/dvirsky/go-pylog/logging"
	"github.com/samuel/go-thrift/examples/scribe"
	"github.com/samuel/go-thrift/thrift"
	//"io"
	//"log"
	"net"
	"os"
	"time"
)

type ScribeLogger struct {
	client          *scribe.ScribeClient
	addr            string
	enabled         bool
	category        string
	channel         chan *scribe.LogEntry
	formatter       logging.Formatter
	secondaryOutput io.Writer
}

func (l *ScribeLogger) connect() error {
	if l.client != nil {
		return nil
	}

	var conn net.Conn
	var err error
	//try to conect 3 times
	for reconns := 0; reconns < _MAX_RETRIES; reconns++ {
		conn, err = net.Dial("tcp", l.addr)
		if err != nil {
			if reconns >= _MAX_RETRIES-1 {
				fmt.Fprintf(os.Stderr, "ERROR: Could not connect to scribe server: %s\n", err)
				return err
			}
			time.Sleep(1000 * time.Millisecond) //wait a bit before retrying
		} else {
			break
		}

	}

	transport := thrift.NewTransport(thrift.NewFramedReadWriteCloser(conn, 0), thrift.BinaryProtocol)
	client := thrift.NewClient(transport, false)
	l.client = &scribe.ScribeClient{Client: client}
	l.enabled = true
	return nil

}

//Create and init a new scribe logger. This will not connect to the scribe server directly
//
// addr is the scribe servers "host:port" string.
//
// category - the scribe category prefix for all your messages. They wil be formatted as "category.LEVEL"
//
// bufferSize is the sending channel's buffer size. 100 is a good estimate. This causes sends to be non blocking
func NewScribeLogger(addr string, category string, bufferSize int, secondaryOutput io.Writer) *ScribeLogger {

	ret := &ScribeLogger{
		addr:            addr,
		client:          nil,
		enabled:         true,
		category:        category,
		channel:         make(chan *scribe.LogEntry, bufferSize),
		formatter:       &ScribeFormatter{},
		secondaryOutput: secondaryOutput,
	}

	go ret.sendLoop()

	return ret
}

const _MAX_RETRIES = 3

// Read from the message channel and send to the scribe server
func (l *ScribeLogger) sendLoop() {

	defer func() {
		e := recover()
		if e != nil {
			fmt.Fprintf(os.Stderr, "Scribe client send loop crashed! restarting...")
			go l.sendLoop()
		}

	}()

	for msg := range l.channel {

		if msg != nil {

			//reconnect or do nothing...
			if l.client == nil {
				e := l.connect()

				if e != nil {
					if l.secondaryOutput == nil {
						fmt.Fprintln(os.Stderr, msg.Message)
					}

					continue
				}
			}

			//send to the server
			_, err := l.client.Log([]*scribe.LogEntry{msg})

			//disconnect if failed
			if err != nil {
				l.client = nil
			}
		}

	}
}

// format for the scribe formatter. we add the hostname so we can distinguish between messages from different servers
const FORMAT_STRING = "[%s] [%s] [%s in %s:%d] %s"

// default host name
var hostName = "localhost"

func init() {
	hostName, _ = os.Hostname()
}

type ScribeFormatter struct{}

func (f *ScribeFormatter) Format(ctx *logging.MessageContext, message string, args ...interface{}) string {

	return fmt.Sprintf(FORMAT_STRING, ctx.TimeStamp.Format(time.StampNano), hostName, ctx.Level, ctx.File, ctx.Line, fmt.Sprintf(message, args...))
}

// Stop the handler from sending to scribe, by setting the enabled flag.
// This will cause the send loop to exit
func (l *ScribeLogger) Stop() {
	l.enabled = false
	close(l.channel)
}

func (l *ScribeLogger) SetFormatter(f logging.Formatter) {
	l.formatter = f
}

// Emit - format the message and queue it to be sent to the scribe server
func (l *ScribeLogger) Emit(ctx *logging.MessageContext, message string, args ...interface{}) error {

	if l.enabled {
		// format the message - we remove the level because scribe already sends them to different pipelines
		// plus we add the timestamp which the default logger already has
		str := l.formatter.Format(ctx, message, args...)
		category := fmt.Sprintf("%s.%s", l.category, ctx.Level)

		if l.secondaryOutput != nil {
			fmt.Fprintln(l.secondaryOutput, str)
		}

		//make sure the channel is not closed
		if l.channel != nil {

			//try sending, aborting immediately if the buffer is full
			select {
			case l.channel <- &scribe.LogEntry{category, str}:
				break
			default: //could not send

				return fmt.Errorf("Scribe buffer full")
			}
		} else {
			return fmt.Errorf("Scribe buffer channel closed")
		}

	}
	return nil
}
