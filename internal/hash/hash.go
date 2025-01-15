//
// @project apfs 2018 - 2019
// @author Dmitry Ponomarev <demdxx@gmail.com> 2018 - 2019
//

package hash

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// FileMd5 from file od disk
func FileMd5(filePath string) (size int64, hashHex string, err error) {
	// Open the passed argument and check for any error
	file, err := os.Open(filePath)
	if err != nil {
		return 0, "", err
	}

	// Tell the program to call the following function when the current function returns
	defer func() { _ = file.Close() }()

	return DataMd5(file)
}

// DataMd5 by reader
func DataMd5(data io.Reader) (size int64, hashHex string, err error) {
	// Open a new hash interface to write to
	hash := md5.New()

	// Copy the data in the hash interface and check for any error
	if size, err = io.Copy(hash, data); err != nil {
		return size, "", err
	}

	// Convert the bytes to a string
	return size, hex.EncodeToString(hash.Sum(nil)), nil
}

// Md5 from data
func Md5(data []byte) (string, error) {
	// Open a new hash interface to write to
	hash := md5.New()
	if _, err := hash.Write(data); err != nil {
		return "", err
	}

	// Convert the bytes to a string
	return hex.EncodeToString(hash.Sum(nil)), nil
}
