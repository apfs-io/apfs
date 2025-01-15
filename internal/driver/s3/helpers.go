package s3

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"

	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/internal/object"
)

func isNotExist(err error) bool {
	if err == nil {
		return false
	}
	if os.IsNotExist(err) {
		return true
	}
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case s3.ErrCodeNoSuchBucket, s3.ErrCodeNoSuchKey:
			return true
		}
	}
	return false
}

func objectKey(object npio.Object, name string) string {
	return filepath.Join(object.Path(), object.PrepareName(name))
}

// NewObject creates basic object from Bucket name + Object Path
// bucket name: images, videos, documents, etc.
// object path: generated bucket full path without bucket => {{year}}/{{month}}/{{md5:1}}/{{md5:2}}/{{md5}}
func newObject(bucket, pathName string) npio.Object {
	return object.NewObject(npio.ObjectIDType(
		strings.Trim(bucket, "/")+"/"+strings.Trim(pathName, "/"),
	), bucket, pathName)
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
