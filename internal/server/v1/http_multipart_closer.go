package v1

import (
	"io"
	"mime/multipart"
)

type multipartCloser struct {
	filePart *multipart.Part
	reader   *multipart.Reader
}

func (m *multipartCloser) Close() error {
	if m.filePart != nil {
		if err := m.filePart.Close(); err != nil {
			return err
		}
	}
	if m.reader != nil {
		for {
			filePart, err := m.reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if err := filePart.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}
