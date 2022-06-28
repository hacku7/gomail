// +build go1.5

package mime

import (
	"mime"
	"mime/quotedprintable"
	"strings"
)

var NewQPWriter = quotedprintable.NewWriter

// MimeEncoder encode
type MimeEncoder struct {
	mime.WordEncoder
}

var (
	BEncoding     = MimeEncoder{mime.BEncoding}
	QEncoding     = MimeEncoder{mime.QEncoding}
	LastIndexByte = strings.LastIndexByte
)
