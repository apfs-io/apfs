package utils

import (
	"image"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	datalib "github.com/apfs-io/apfs/internal/storage/data"
	"github.com/apfs-io/apfs/models"
)

// CollectFileInfo process and fill base file information icluding media information
func CollectFileInfo(meta *models.ItemMeta, fileFullpath, contentType string) (_ *models.ItemMeta, err error) {
	file, err := os.Open(fileFullpath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return CollectReadSeekerInfo(meta, file, fileFullpath, contentType)
}

// CollectReadSeekerInfo process and fill base buffer information icluding media information
func CollectReadSeekerInfo(meta *models.ItemMeta, reader io.ReadSeeker, fileFullpath, contentType string) (_ *models.ItemMeta, err error) {
	if contentType == "" && meta != nil {
		contentType = meta.ContentType
	}

	// Detect content type if no defined
	if contentType == "" {
		if contentType, err = datalib.ContentTypeByReadSeeker(reader); err != nil {
			return nil, err
		}
	}

	var (
		hashID        string
		basename      = filepath.Base(fileFullpath)
		fileExt       = filepath.Ext(fileFullpath)
		objectType    = models.ObjectTypeByContentType(contentType)
		width, height int
	)

	if fileExt == "" {
		if types, _ := mime.ExtensionsByType(contentType); len(types) > 0 {
			fileExt = types[0]
		}
	}

	if meta == nil {
		meta = &models.ItemMeta{}
	}

	// Calculate MD5 hash
	if meta.Size, hashID, err = datalib.HashDataMd5(reader); err != nil {
		return nil, err
	}

	// Reset position of the file object
	if _, err = reader.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	if objectType.IsImage() {
		width, height, err = imageSizeByReader(reader)
	}

	meta.Name = basename
	meta.NameExt = strings.ToLower(strings.TrimLeft(fileExt, "."))
	meta.Type = objectType
	meta.ContentType = contentType
	meta.HashID = hashID
	meta.Width = width
	meta.Height = height

	return meta, err
}

func imageSizeByReader(reader io.Reader) (w, h int, err error) {
	var conf image.Config
	if conf, _, err = image.DecodeConfig(reader); err == nil {
		if seeker, _ := reader.(io.ReadSeeker); seeker != nil {
			_, err = seeker.Seek(0, io.SeekStart)
		}
	}
	return conf.Width, conf.Height, err
}

func trimExt(baseName string) string {
	if !strings.ContainsAny(baseName, ".") {
		return baseName
	}
	nameExt := filepath.Ext(baseName)
	if nameExt != "" {
		baseName = baseName[:len(baseName)-len(nameExt)]
	}
	return baseName
}
