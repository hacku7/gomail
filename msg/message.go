package msg

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/hacku7/gomail/mime"
	"github.com/hacku7/gomail/writer"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type Header map[string][]string

// Encoding represents a MIME encoding scheme like quoted-printable or base64.
type Encoding string

const (
	// QuotedPrintable represents the quoted-printable encoding as defined in
	// RFC 2045.
	QuotedPrintable Encoding = "quoted-printable"
	// Base64 represents the base64 encoding as defined in RFC 2045.
	Base64 Encoding = "base64"
	// Unencoded can be used to avoid encoding the body of an email. The headers
	// will still be encoded using quoted-printable encoding.
	Unencoded Encoding = "8bit"
)

type Part struct {
	ContentType string
	Copier      func(io.Writer) error
	Encoding    Encoding
}

// Message represents an email.
type Message struct {
	Header      Header
	Parts       []*Part
	Attachments []*File
	Embedded    []*File
	Charset     string
	Encoding    Encoding
	HEncoder    mime.MimeEncoder
	Buf         bytes.Buffer
}

// NewMessage creates a new msg. It uses UTF-8 and quoted-printable encoding
// by default.
func NewMessage(settings ...MessageSetting) *Message {
	m := &Message{
		Header:   make(Header),
		Charset:  "UTF-8",
		Encoding: QuotedPrintable,
	}

	m.applySettings(settings)

	if m.Encoding == Base64 {
		m.HEncoder = mime.BEncoding
	} else {
		m.HEncoder = mime.QEncoding
	}

	return m
}

// Reset resets the msg so it can be reused. The msg keeps its previous
// settings so it is in the same state that after a call to NewMessage.
func (m *Message) Reset() {
	for k := range m.Header {
		delete(m.Header, k)
	}
	m.Parts = nil
	m.Attachments = nil
	m.Embedded = nil
}

func (m *Message) applySettings(settings []MessageSetting) {
	for _, s := range settings {
		s(m)
	}
}

// A MessageSetting can be used as an argument in NewMessage to configure an
// email.
type MessageSetting func(m *Message)

// SetCharset is a msg setting to set the charset of the email.
func SetCharset(charset string) MessageSetting {
	return func(m *Message) {
		m.Charset = charset
	}
}

// SetEncoding is a msg setting to set the encoding of the email.
func SetEncoding(enc Encoding) MessageSetting {
	return func(m *Message) {
		m.Encoding = enc
	}
}

// SetHeader sets a value to the given header field.
func (m *Message) SetHeader(field string, value ...string) {
	m.encodeHeader(value)
	m.Header[field] = value
}

func (m *Message) encodeHeader(values []string) {
	for i := range values {
		values[i] = m.encodeString(values[i])
	}
}

func (m *Message) encodeString(value string) string {
	return m.HEncoder.Encode(m.Charset, value)
}

// SetHeaders sets the msg headers.
func (m *Message) SetHeaders(h map[string][]string) {
	for k, v := range h {
		m.SetHeader(k, v...)
	}
}

// SetAddressHeader sets an address to the given header field.
func (m *Message) SetAddressHeader(field, address, name string) {
	m.Header[field] = []string{m.FormatAddress(address, name)}
}

// FormatAddress formats an address and a name as a valid RFC 5322 address.
func (m *Message) FormatAddress(address, name string) string {
	if name == "" {
		return address
	}

	enc := m.encodeString(name)
	if enc == name {
		m.Buf.WriteByte('"')
		for i := 0; i < len(name); i++ {
			b := name[i]
			if b == '\\' || b == '"' {
				m.Buf.WriteByte('\\')
			}
			m.Buf.WriteByte(b)
		}
		m.Buf.WriteByte('"')
	} else if hasSpecials(name) {
		m.Buf.WriteString(mime.BEncoding.Encode(m.Charset, name))
	} else {
		m.Buf.WriteString(enc)
	}
	m.Buf.WriteString(" <")
	m.Buf.WriteString(address)
	m.Buf.WriteByte('>')

	addr := m.Buf.String()
	m.Buf.Reset()
	return addr
}

func hasSpecials(text string) bool {
	for i := 0; i < len(text); i++ {
		switch c := text[i]; c {
		case '(', ')', '<', '>', '[', ']', ':', ';', '@', '\\', ',', '.', '"':
			return true
		}
	}

	return false
}

// SetDateHeader sets a date to the given header field.
func (m *Message) SetDateHeader(field string, date time.Time) {
	m.Header[field] = []string{m.FormatDate(date)}
}

// FormatDate formats a date as a valid RFC 5322 date.
func (m *Message) FormatDate(date time.Time) string {
	return date.Format(time.RFC1123Z)
}

// GetHeader gets a header field.
func (m *Message) GetHeader(field string) []string {
	return m.Header[field]
}

// SetBody sets the body of the msg. It replaces any content previously set
// by SetBody, AddAlternative or AddAlternativeWriter.
func (m *Message) SetBody(contentType, body string, settings ...PartSetting) {
	m.Parts = []*Part{m.newPart(contentType, newCopier(body), settings)}
}

// AddAlternative adds an alternative part to the msg.
//
// It is commonly used to send HTML emails that default to the plain text
// version for backward compatibility. AddAlternative appends the new part to
// the end of the msg. So the plain text part should be added before the
// HTML part. See http://en.wikipedia.org/wiki/MIME#Alternative
func (m *Message) AddAlternative(contentType, body string, settings ...PartSetting) {
	m.AddAlternativeWriter(contentType, newCopier(body), settings...)
}

func newCopier(s string) func(io.Writer) error {
	return func(w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	}
}

// AddAlternativeWriter adds an alternative part to the msg. It can be
// useful with the text/template or html/template packages.
func (m *Message) AddAlternativeWriter(contentType string, f func(io.Writer) error, settings ...PartSetting) {
	m.Parts = append(m.Parts, m.newPart(contentType, f, settings))
}

func (m *Message) newPart(contentType string, f func(io.Writer) error, settings []PartSetting) *Part {
	p := &Part{
		ContentType: contentType,
		Copier:      f,
		Encoding:    m.Encoding,
	}

	for _, s := range settings {
		s(p)
	}

	return p
}

// A PartSetting can be used as an argument in Message.SetBody,
// Message.AddAlternative or Message.AddAlternativeWriter to configure the part
// added to a msg.
type PartSetting func(*Part)

// SetPartEncoding sets the encoding of the part added to the msg. By
// default, parts use the same encoding than the msg.
func SetPartEncoding(e Encoding) PartSetting {
	return PartSetting(func(p *Part) {
		p.Encoding = e
	})
}

type File struct {
	Name     string
	Header   map[string][]string
	CopyFunc func(w io.Writer) error
}

func (f *File) SetHeader(field, value string) {
	f.Header[field] = []string{value}
}

// A FileSetting can be used as an argument in Message.Attach or Message.Embed.
type FileSetting func(*File)

// SetHeader is a file setting to set the MIME header of the msg part that
// contains the file content.
//
// Mandatory headers are automatically added if they are not set when sending
// the email.
func SetHeader(h map[string][]string) FileSetting {
	return func(f *File) {
		for k, v := range h {
			f.Header[k] = v
		}
	}
}

// Rename is a file setting to set the name of the attachment if the name is
// different than the filename on disk.
func Rename(name string) FileSetting {
	return func(f *File) {
		f.Name = name
	}
}

// SetCopyFunc is a file setting to replace the function that runs when the
// msg is sent. It should copy the content of the file to the io.Writer.
//
// The default copy function opens the file with the given filename, and copy
// its content to the io.Writer.
func SetCopyFunc(f func(io.Writer) error) FileSetting {
	return func(fi *File) {
		fi.CopyFunc = f
	}
}

func (m *Message) appendFile(list []*File, name string, settings []FileSetting) []*File {
	f := &File{
		Name:   filepath.Base(name),
		Header: make(map[string][]string),
		CopyFunc: func(w io.Writer) error {
			h, err := os.Open(name)
			if err != nil {
				return err
			}
			if _, err := io.Copy(w, h); err != nil {
				h.Close()
				return err
			}
			return h.Close()
		},
	}

	for _, s := range settings {
		s(f)
	}

	if list == nil {
		return []*File{f}
	}

	return append(list, f)
}

// Attach attaches the files to the email.
func (m *Message) Attach(filename string, settings ...FileSetting) {
	m.Attachments = m.appendFile(m.Attachments, filename, settings)
}

// Embed embeds the images to the email.
func (m *Message) Embed(filename string, settings ...FileSetting) {
	m.Embedded = m.appendFile(m.Embedded, filename, settings)
}

func (m *Message) HasMixedPart() bool {
	return (len(m.Parts) > 0 && len(m.Attachments) > 0) || len(m.Attachments) > 1
}

func (m *Message) HasRelatedPart() bool {
	return (len(m.Parts) > 0 && len(m.Embedded) > 0) || len(m.Embedded) > 1
}

func (m *Message) HasAlternativePart() bool {
	return len(m.Parts) > 1
}

// WriteTo implements io.WriterTo. It dumps the whole msg into w.
func (m *Message) WriteTo(w io.Writer) (int64, error) {
	mw := &writer.MessageWriter{W: w}
	mw.WriteMessage(m)
	return mw.N, mw.Err
}

func (m *Message) GetFrom() (string, error) {
	from := m.Header["Sender"]
	if len(from) == 0 {
		from = m.Header["From"]
		if len(from) == 0 {
			return "", errors.New(`gomail: invalid msg, "From" field is absent`)
		}
	}

	return parseAddress(from[0])
}

func (m *Message) GetRecipients() ([]string, error) {
	n := 0
	for _, field := range []string{"To", "Cc", "Bcc"} {
		if addresses, ok := m.Header[field]; ok {
			n += len(addresses)
		}
	}
	list := make([]string, 0, n)

	for _, field := range []string{"To", "Cc", "Bcc"} {
		if addresses, ok := m.Header[field]; ok {
			for _, a := range addresses {
				addr, err := parseAddress(a)
				if err != nil {
					return nil, err
				}
				list = addAddress(list, addr)
			}
		}
	}

	return list, nil
}

func CompareBodies(t *testing.T, got, want string) {
	// We cannot do a simple comparison since the ordering of headers' fields
	// is random.
	gotLines := strings.Split(got, "\r\n")
	wantLines := strings.Split(want, "\r\n")

	// We only test for too many lines, missing lines are tested after
	if len(gotLines) > len(wantLines) {
		t.Fatalf("Message has too many lines, \ngot %d:\n%s\nwant %d:\n%s", len(gotLines), got, len(wantLines), want)
	}

	isInHeader := true
	headerStart := 0
	for i, line := range wantLines {
		if line == gotLines[i] {
			if line == "" {
				isInHeader = false
			} else if !isInHeader && len(line) > 2 && line[:2] == "--" {
				isInHeader = true
				headerStart = i + 1
			}
			continue
		}

		if !isInHeader {
			missingLine(t, line, got, want)
		}

		isMissing := true
		for j := headerStart; j < len(gotLines); j++ {
			if gotLines[j] == "" {
				break
			}
			if gotLines[j] == line {
				isMissing = false
				break
			}
		}
		if isMissing {
			missingLine(t, line, got, want)
		}
	}
}

func missingLine(t *testing.T, line, got, want string) {
	t.Fatalf("Missing line %q\ngot:\n%s\nwant:\n%s", line, got, want)
}

func addAddress(list []string, addr string) []string {
	for _, a := range list {
		if addr == a {
			return list
		}
	}

	return append(list, addr)
}

func parseAddress(field string) (string, error) {
	addr, err := mail.ParseAddress(field)
	if err != nil {
		return "", fmt.Errorf("gomail: invalid address %q: %v", field, err)
	}
	return addr.Address, nil
}
