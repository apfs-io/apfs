package s3

import (
	"errors"
	"testing"
)

func TestIsNotExistAWSNoSuchKey(t *testing.T) {
	err := errors.New("operation error S3: GetObject, https response error StatusCode: 404, RequestID: 18D0542B0EEDB52A, HostID: dd9025bab4ad464b049177c95eb6ebf374d3b3fd1af9251148b658df7ac2e3e8, NoSuchKey: The specified key does not exist.")
	if !isNotExist(err) {
		t.Fatal("expected isNotExist for AWS SDK v2 GetObject 404 NoSuchKey")
	}
}
