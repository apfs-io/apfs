package s3

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/internal/object"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

func isNotExist(err error) bool {
	if err == nil {
		return false
	}
	if os.IsNotExist(err) {
		return true
	}
	var respErr *smithyhttp.ResponseError
	if errors.As(err, &respErr) {
		if respErr.HTTPStatusCode() == 404 {
			return true
		}
	}
	switch errorCode(err) {
	case "NoSuchBucket", "NoSuchKey":
		return true
	}
	return false
}

func errorCode(err error) string {
	if err == nil {
		return ""
	}
	unErr := errors.Unwrap(err)
	if code := _errorCode(err); code != "" {
		return code
	}
	if unErr != nil {
		if code := _errorCode(unErr); code != "" {
			return code
		}
	}
	return ""
}

func _errorCode(err error) string {
	if err == nil {
		return ""
	}
	type errorCode interface {
		ErrorCode() string
	}

	// Check if the error directly implements the errorCode interface
	if apiErr, _ := err.(errorCode); apiErr != nil {
		return apiErr.ErrorCode()
	}
	return ""
}

func objectKey(object npio.Object, name string) string {
	return filepath.Join(object.Path(), object.PrepareName(name))
}

// NewObject creates basic object from Bucket name + Object Path
// bucket name: images, videos, documents, etc.
// object path: generated bucket full path without bucket => {{year}}/{{month}}/{{md5:1}}/{{md5:2}}/{{md5}}
func newObject(bucket, pathName string) npio.Object {
	return object.NewObject(newObjectID(bucket, pathName), bucket, pathName)
}

func newObjectID(bucket, pathName string) npio.ObjectIDType {
	return npio.ObjectIDType(
		strings.Trim(bucket, "/") + "/" + strings.Trim(pathName, "/"),
	)
}

func objectFromID(id npio.ObjectID) npio.Object {
	filepath := string(id.ID())
	splits := strings.SplitN(filepath, "/", 2)
	if len(splits) != 2 {
		return object.NewObject("", "", "")
	}
	return object.NewObject(id.ID(), splits[0], splits[1])
}

func strIf(first bool, s1, s2 string) string {
	if first {
		return s1
	}
	return s2
}

func closeObject(obj any) error {
	switch c := obj.(type) {
	case nil:
	case io.Closer:
		return c.Close()
	}
	return nil
}
