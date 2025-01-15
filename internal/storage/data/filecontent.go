//
// @project apfs 2018
// @author Dmitry Ponomarev <demdxx@gmail.com> 2018
//

package data

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const fileHeadSize = 10 * 1024

// ExtensionByContentType for the file
func ExtensionByContentType(contentType string) string {
	switch contentType {
	case "text/plain":
		return ".txt"
	case "text/html", "text/xhtml":
		return ".html"
	case "text/xml", "application/xml":
		return ".xml"
	case "text/json", "application/json":
		return ".json"
	case "text/javascript", "application/javascript":
		return ".js"
	}

	exts, _ := mime.ExtensionsByType(contentType)
	if len(exts) > 0 {
		switch exts[0] {
		case ".jpe":
			return ".jpg"
		default:
			return exts[0]
		}
	}
	return ""
}

// CopyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func CopyFileContents(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	return SaveDataToFile(in, dst)
}

// SaveDataToFile on disk
func SaveDataToFile(data io.Reader, dst string) (err error) {
	// Create directory path
	if dir := filepath.Dir(dst); len(dir) > 0 {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return
		}
	}

	out, err := os.Create(dst)
	if err != nil {
		return
	}

	defer func() {
		if cerr := out.Close(); err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(out, data); err != nil {
		return
	}

	err = out.Sync()
	return
}

// ReadHead from reader
func ReadHead(data io.Reader, size ...int) (head []byte, rd io.ReadCloser, err error) {
	var (
		bytesCount = fileHeadSize
		l          int
	)

	if len(size) > 0 && size[0] > 0 {
		bytesCount = size[0]
	}

	head = make([]byte, bytesCount)
	if l, err = data.Read(head); err != nil {
		if err != io.EOF {
			return
		}
		err = nil
	}

	head = head[:l]
	rd = NewGroupReader(bytes.NewReader(head), data)

	return head, rd, err
}

// ContentTypeByReadSeeker detection
func ContentTypeByReadSeeker(data io.ReadSeeker) (contentType string, err error) {
	var (
		head     = make([]byte, fileHeadSize)
		headSize int
	)
	if headSize, err = data.Read(head); err != nil {
		if err != io.EOF {
			return "", err
		}
	}
	head = head[:headSize]
	if _, err = data.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return ContentTypeByData(head), nil
}

// ContentTypeByData detection
func ContentTypeByData(head []byte) (contentType string) {
	contentType = http.DetectContentType(head)
	if i := strings.IndexRune(contentType, ';'); i >= 0 {
		contentType = contentType[:i]
	}
	return contentType
}
