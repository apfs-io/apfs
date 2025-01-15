//
// @project apfs 2018
// @author Dmitry Ponomarev <demdxx@gmail.com> 2018
//

package storage

import (
	"os"

	"github.com/apfs-io/apfs/models"
)

// DB basic accessor
type DB interface {
	Get(id string) (*models.Object, error)
	Set(obj *models.Object) error
	Delete(id string) error
}

// DatabaseMock object
type DatabaseMock struct{}

// Get file base object
func (db *DatabaseMock) Get(id string) (*models.Object, error) {
	return nil, os.ErrNotExist
}

// Set file base object
func (db *DatabaseMock) Set(obj *models.Object) error {
	return nil
}

// Delete file base object
func (db *DatabaseMock) Delete(id string) error {
	return nil
}
