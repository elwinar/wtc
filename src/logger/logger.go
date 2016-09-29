package logger

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/go-logfmt/logfmt"
)

type M map[string]interface{}

// A Logger is a simple structured logger implementation using logfmt format.
type Logger struct {
	out io.Writer
	ctx M
}

// New returns a new logger which will write to the given writer.
func New(out io.Writer) *Logger {
	if out == nil {
		out = os.Stdout
	}

	return &Logger{
		out: out,
	}
}

// With returns a new logger with an augmented context.
func (l *Logger) With(ctx M) *Logger {
	// Create a new context for the new logger.
	c := M{}

	// Add both the old context and the new additions.
	for _, m := range []M{l.ctx, ctx} {
		if m == nil {
			continue
		}
		for k, v := range m {
			c[k] = v
		}
	}

	// Return a new logger with the new context.
	return &Logger{
		out: l.out,
		ctx: c,
	}
}

// Log writes a log line to the output of the logger with the level, message
// and data in parameters.
func (l *Logger) Log(lvl, msg string, data M) {
	// Ensure the map is initialized.
	if data == nil {
		data = M{}
	}

	// Add the fields from the context.
	if l.ctx != nil {
		for k, v := range l.ctx {
			data[k] = v
		}
	}

	// Add the par-line fields.
	data["lvl"] = lvl
	data["msg"] = msg
	data["time"] = time.Now().Format(time.RFC3339)

	// Get each resulting field and sort them.
	fields := make([]string, 0, len(data))
	for f := range data {
		fields = append(fields, f)
	}
	sort.Strings(fields)

	// Write them on the buffer.
	var buf bytes.Buffer
	var enc = logfmt.NewEncoder(&buf)
	for _, f := range fields {
		_ = enc.EncodeKeyval(f, fmt.Sprint(data[f]))
	}
	_ = enc.EndRecord()

	// Write the buffer on the output.
	_, _ = l.out.Write(buf.Bytes())
}

// Error is a shortcut to write an error log line.
func (l *Logger) Error(msg string, data M) {
	l.Log("error", msg, data)
}

// Info is a shortcut to write an info log line.
func (l *Logger) Info(msg string, data M) {
	l.Log("info", msg, data)
}

// SetOutput allows to change the output of the logger.
func (l *Logger) SetOutput(out io.Writer) {
	l.out = out
}
