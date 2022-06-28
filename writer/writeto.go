package writer

import (
	"encoding/base64"
	"errors"
	mime1 "github.com/hacku7/gomail/mime"
	"github.com/hacku7/gomail/msg"
	"io"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"
)

func (w *MessageWriter) WriteMessage(m *msg.Message) {
	if _, ok := m.Header["Mime-Version"]; !ok {
		w.writeString("Mime-Version: 1.0\r\n")
	}
	if _, ok := m.Header["Date"]; !ok {
		w.writeHeader("Date", m.FormatDate(Now()))
	}
	w.writeHeaders(m.Header)

	if m.HasMixedPart() {
		w.openMultipart("mixed")
	}

	if m.HasRelatedPart() {
		w.openMultipart("related")
	}

	if m.HasAlternativePart() {
		w.openMultipart("alternative")
	}
	for _, part := range m.Parts {
		w.writePart(part, m.Charset)
	}
	if m.HasAlternativePart() {
		w.closeMultipart()
	}

	w.addFiles(m.Embedded, false)
	if m.HasRelatedPart() {
		w.closeMultipart()
	}

	w.addFiles(m.Attachments, true)
	if m.HasMixedPart() {
		w.closeMultipart()
	}
}

type MessageWriter struct {
	W          io.Writer
	N          int64
	Writers    [3]*multipart.Writer
	PartWriter io.Writer
	Depth      uint8
	Err        error
}

func (w *MessageWriter) openMultipart(mimeType string) {
	mw := multipart.NewWriter(w)
	contentType := "multipart/" + mimeType + ";\r\n boundary=" + mw.Boundary()
	w.Writers[w.Depth] = mw

	if w.Depth == 0 {
		w.writeHeader("Content-Type", contentType)
		w.writeString("\r\n")
	} else {
		w.createPart(map[string][]string{
			"Content-Type": {contentType},
		})
	}
	w.Depth++
}

func (w *MessageWriter) createPart(h map[string][]string) {
	w.PartWriter, w.Err = w.Writers[w.Depth-1].CreatePart(h)
}

func (w *MessageWriter) closeMultipart() {
	if w.Depth > 0 {
		w.Writers[w.Depth-1].Close()
		w.Depth--
	}
}

func (w *MessageWriter) writePart(p *msg.Part, charset string) {
	w.writeHeaders(map[string][]string{
		"Content-Type":              {p.ContentType + "; charset=" + charset},
		"Content-Transfer-Encoding": {string(p.Encoding)},
	})
	w.writeBody(p.Copier, p.Encoding)
}

func (w *MessageWriter) addFiles(files []*msg.File, isAttachment bool) {
	for _, f := range files {
		if _, ok := f.Header["Content-Type"]; !ok {
			mediaType := mime.TypeByExtension(filepath.Ext(f.Name))
			if mediaType == "" {
				mediaType = "application/octet-stream"
			}
			f.SetHeader("Content-Type", mediaType+`; name="`+f.Name+`"`)
		}

		if _, ok := f.Header["Content-Transfer-Encoding"]; !ok {
			f.SetHeader("Content-Transfer-Encoding", string(msg.Base64))
		}

		if _, ok := f.Header["Content-Disposition"]; !ok {
			var disp string
			if isAttachment {
				disp = "attachment"
			} else {
				disp = "inline"
			}
			f.SetHeader("Content-Disposition", disp+`; filename="`+f.Name+`"`)
		}

		if !isAttachment {
			if _, ok := f.Header["Content-ID"]; !ok {
				f.SetHeader("Content-ID", "<"+f.Name+">")
			}
		}
		w.writeHeaders(f.Header)
		w.writeBody(f.CopyFunc, msg.Base64)
	}
}

func (w *MessageWriter) Write(p []byte) (int, error) {
	if w.Err != nil {
		return 0, errors.New("gomail: cannot write as writer is in error")
	}

	var n int
	n, w.Err = w.W.Write(p)
	w.N += int64(n)
	return n, w.Err
}

func (w *MessageWriter) writeString(s string) {
	n, _ := io.WriteString(w.W, s)
	w.N += int64(n)
}

func (w *MessageWriter) writeHeader(k string, v ...string) {
	w.writeString(k)
	if len(v) == 0 {
		w.writeString(":\r\n")
		return
	}
	w.writeString(": ")

	// Max header line length is 78 characters in RFC 5322 and 76 characters
	// in RFC 2047. So for the sake of simplicity we use the 76 characters
	// limit.
	charsLeft := 76 - len(k) - len(": ")

	for i, s := range v {
		// If the line is already too long, insert a newline right away.
		if charsLeft < 1 {
			if i == 0 {
				w.writeString("\r\n ")
			} else {
				w.writeString(",\r\n ")
			}
			charsLeft = 75
		} else if i != 0 {
			w.writeString(", ")
			charsLeft -= 2
		}

		// While the header content is too long, fold it by inserting a newline.
		for len(s) > charsLeft {
			s = w.writeLine(s, charsLeft)
			charsLeft = 75
		}
		w.writeString(s)
		if i := mime1.LastIndexByte(s, '\n'); i != -1 {
			charsLeft = 75 - (len(s) - i - 1)
		} else {
			charsLeft -= len(s)
		}
	}
	w.writeString("\r\n")
}

func (w *MessageWriter) writeLine(s string, charsLeft int) string {
	// If there is already a newline before the limit. Write the line.
	if i := strings.IndexByte(s, '\n'); i != -1 && i < charsLeft {
		w.writeString(s[:i+1])
		return s[i+1:]
	}

	for i := charsLeft - 1; i >= 0; i-- {
		if s[i] == ' ' {
			w.writeString(s[:i])
			w.writeString("\r\n ")
			return s[i+1:]
		}
	}

	// We could not insert a newline cleanly so look for a space or a newline
	// even if it is after the limit.
	for i := 75; i < len(s); i++ {
		if s[i] == ' ' {
			w.writeString(s[:i])
			w.writeString("\r\n ")
			return s[i+1:]
		}
		if s[i] == '\n' {
			w.writeString(s[:i+1])
			return s[i+1:]
		}
	}

	// Too bad, no space or newline in the whole string. Just write everything.
	w.writeString(s)
	return ""
}

func (w *MessageWriter) writeHeaders(h map[string][]string) {
	if w.Depth == 0 {
		for k, v := range h {
			if k != "Bcc" {
				w.writeHeader(k, v...)
			}
		}
	} else {
		w.createPart(h)
	}
}

func (w *MessageWriter) writeBody(f func(io.Writer) error, enc msg.Encoding) {
	var subWriter io.Writer
	if w.Depth == 0 {
		w.writeString("\r\n")
		subWriter = w.W
	} else {
		subWriter = w.PartWriter
	}

	if enc == msg.Base64 {
		wc := base64.NewEncoder(base64.StdEncoding, newBase64LineWriter(subWriter))
		w.Err = f(wc)
		wc.Close()
	} else if enc == msg.Unencoded {
		w.Err = f(subWriter)
	} else {
		wc := mime1.NewQPWriter(subWriter)
		w.Err = f(wc)
		wc.Close()
	}
}

// As required by RFC 2045, 6.7. (page 21) for quoted-printable, and
// RFC 2045, 6.8. (page 25) for base64.
const maxLineLen = 76

// base64LineWriter limits text encoded in base64 to 76 characters per line
type base64LineWriter struct {
	w       io.Writer
	lineLen int
}

func newBase64LineWriter(w io.Writer) *base64LineWriter {
	return &base64LineWriter{w: w}
}

func (w *base64LineWriter) Write(p []byte) (int, error) {
	n := 0
	for len(p)+w.lineLen > maxLineLen {
		w.w.Write(p[:maxLineLen-w.lineLen])
		w.w.Write([]byte("\r\n"))
		p = p[maxLineLen-w.lineLen:]
		n += maxLineLen - w.lineLen
		w.lineLen = 0
	}

	w.w.Write(p)
	w.lineLen += len(p)

	return n + len(p), nil
}

// Stubbed out for testing.
var Now = time.Now
