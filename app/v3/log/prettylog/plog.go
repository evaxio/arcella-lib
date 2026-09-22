package prettylog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	timestampFormat string = "06.01.02 15:04:05.0000"
	//encoding        string = "WINDOWS-1251"
	//logFileName     string = "app.log"
)

type IndentHandler struct {
	opts    Options
	goas    []groupOrAttrs
	mu      *sync.Mutex
	out     io.Writer
	withGID bool
}

type Options struct {
	// Level reports the minimum level to log.
	// Levels with lower levels are discarded.
	// If nil, the Handler uses [slog.LevelInfo].
	Level slog.Leveler
}

// groupOrAttrs holds either a group name or a list of slog.Attrs.
type groupOrAttrs struct {
	group string      // group name if non-empty
	attrs []slog.Attr // attrs if non-empty
}

func New(out io.Writer, opts *Options) *IndentHandler {
	h := &IndentHandler{out: out, mu: &sync.Mutex{}, withGID: true}
	if opts != nil {
		h.opts = *opts
	}
	if h.opts.Level == nil {
		h.opts.Level = slog.LevelDebug
	}
	return h
}

func (h *IndentHandler) WithoutGID() *IndentHandler {
	h2 := *h
	h2.withGID = false
	return &h2
}

func (h *IndentHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *IndentHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}
func (h *IndentHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}

func (h *IndentHandler) withGroupOrAttrs(goa groupOrAttrs) *IndentHandler {
	h2 := *h
	h2.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(h2.goas, h.goas)
	if len(h2.goas)-1 >= 0 {
		h2.goas[len(h2.goas)-1] = goa
	}
	return &h2
}

func (h *IndentHandler) writeGID(buf *buffer) {
	if !h.withGID {
		return
	}
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)] // goroutine 1 [running]: ...
	b = b[10:]                      // "goroutine "
	if i := bytes.IndexByte(b, ' '); i >= 0 {
		b = b[:i]
	}
	buf.WriteByte(' ')
	buf.WriteByte('[')
	buf.Write(b)
	buf.WriteByte(']')
	buf.WriteByte(' ')
}

type funcDataEntry struct {
	file     string
	funcName string
	line     string
}

var (
	funcDataMu    sync.Mutex
	funcDataCache = make(map[uintptr]funcDataEntry)
)

func (h *IndentHandler) writeFuncData(buf *buffer, r slog.Record) {
	funcDataMu.Lock()
	fde, ok := funcDataCache[r.PC]
	funcDataMu.Unlock()
	if !ok {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()

		// short function name
		name := f.Func.Name()
		if i := strings.LastIndex(name, "."); i >= 0 {
			name = name[i+1:]
		}

		// short file name
		_, fileName := path.Split(f.File)

		fde = funcDataEntry{file: fileName, funcName: name, line: strconv.Itoa(f.Line)}
		funcDataMu.Lock()
		funcDataCache[r.PC] = fde
		funcDataMu.Unlock()
	}

	buf.WriteString(fde.file)
	buf.WriteByte(' ')
	buf.WriteString(fde.funcName)
	buf.WriteByte(':')
	buf.WriteString(fde.line)
}

const maxStrSize = 900

func (h *IndentHandler) Handle(ctx context.Context, r slog.Record) error {
	// get a buffer from the sync pool
	buf := newBuffer()
	defer buf.Free()

	var levelStr string
	switch r.Level {
	case slog.LevelDebug:
		levelStr = "DBG "
	case slog.LevelInfo:
		levelStr = "INF "
	case slog.LevelWarn:
		levelStr = "WRN "
	case slog.LevelError:
		levelStr = "ERR "
	default:
		levelStr = strconv.FormatInt(int64(r.Level), 10) + " "
	}
	buf.WriteString(levelStr)
	buf.WriteString(r.Time.Format(timestampFormat))
	h.writeGID(buf)
	if r.PC != 0 {
		h.writeFuncData(buf, r)
	}

	// message
	buf.WriteString(" - ")
	if len(r.Message) > maxStrSize {
		// truncate at a rune boundary
		end := maxStrSize
		for end > 0 && !utf8.RuneStart(r.Message[end]) {
			end--
		}
		buf.WriteString(r.Message[:end])
		buf.WriteString("...")
	} else {
		buf.WriteString(r.Message)
	}

	// Groups And Attrs
	h.writeGroups(buf, r)
	h.writeAttrData(buf, r)

	_ = buf.WriteByte('\n')
	h.mu.Lock()
	_, err := h.out.Write(*buf)
	h.mu.Unlock()
	return err
}

func (h *IndentHandler) writeGroups(buf *buffer, r slog.Record) {
	for _, goa := range h.goas {
		if goa.group != "" {
			buf.WriteString(" ")
			buf.WriteString(goa.group)
			buf.WriteString(": ")
		}
		if len(goa.attrs) > 0 {
			buf.WriteByte('[')
			for _, attr := range goa.attrs {
				buf.WriteString(attr.Key)
				buf.WriteByte('=')
				buf.WriteString(attr.Value.String())
			}
			buf.WriteString("] ")
		}
	}
}

func (h *IndentHandler) writeAttrData(buf *buffer, r slog.Record) {
	if r.NumAttrs() > 0 {
		_ = buf.WriteByte(' ')
		r.Attrs(func(a slog.Attr) bool {
			buf.WriteString(a.Key)
			buf.WriteByte('=')
			switch a.Value.Kind() {
			case slog.KindAny:
				if r.Level == slog.LevelDebug && reflect.TypeOf(a.Value.Any()).Kind() == reflect.Struct {
					if b, err := json.Marshal(a.Value.Any()); err == nil {
						buf.Write(b)
					} else {
						buf.WriteString(fmt.Sprintf("%+v", a.Value.Any()))
					}
				} else {
					buf.WriteString(fmt.Sprintf("%+v", a.Value.Any()))
				}
			default:
				buf.WriteString(a.Value.String())
			}
			_ = buf.WriteByte(' ')
			return true
		})
	}
}
