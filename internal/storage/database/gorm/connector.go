package gorm

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/demdxx/gocast/v2"
	"github.com/jinzhu/gorm"

	"github.com/apfs-io/apfs/internal/storage"
	"github.com/apfs-io/apfs/models"
)

// Connector object
type connector struct {
	conn *gorm.DB
}

// Connect database by URL
func Connect(connectURL string) (storage.DB, error) {
	u, err := url.Parse(connectURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "sqlite3" || u.Scheme == "sqlite" {
		connectURL = strings.TrimPrefix(connectURL, "sqlite3://")
	}
	return New(u.Scheme, connectURL,
		gocast.Bool(u.Query().Get(`automigrate`)),
		gocast.Bool(u.Query().Get(`debug`)))
}

// New db connector
func New(dialect, connect string, automigrate, debug bool) (*connector, error) {
	conn, err := gorm.Open(dialect, connect)
	if err != nil {
		return nil, err
	}

	if debug {
		conn = conn.Debug()
	}

	switch dialect {
	case "postgres":
		conn.Exec(`CREATE TABLE IF NOT EXISTS object (
      path            TEXT          PRIMARY KEY
    , hashid          VARCHAR(128)  NOT NULL
    , content_type    VARCHAR(64)   NOT NULL
    , type            VARCHAR(32)   NOT NULL
    , tags            TEXT[]
    , meta            JSONB
    , size            BIGINT        NOT NULL  DEFAULT 0
    , created_at      TIMESTAMPTZ   NOT NULL  DEFAULT NOW()
    , updated_at      TIMESTAMPTZ   NOT NULL  DEFAULT NOW()
		)`)
	case "mysql":
		conn = conn.Set("gorm:table_options", "ENGINE=InnoDB")
	case "sqlite3":
		conn.AutoMigrate(&models.Object{})
	default:
		return nil, fmt.Errorf("unsupported database: %s", connect)
	}
	if automigrate && dialect != "sqlite3" {
		conn.AutoMigrate(&models.Object{})
	}
	return &connector{conn: conn}, nil
}

// Get file base object
func (db *connector) Get(objID string) (*models.Object, error) {
	var (
		obj models.Object
		res = db.conn.Where("path = ?", objID).Find(&obj)
	)
	return &obj, res.Error
}

// Set file base object
func (db *connector) Set(obj *models.Object) error {
	var (
		count int
		res   *gorm.DB
	)
	res = db.conn.Model((*models.Object)(nil)).
		Where("path = ?", obj.ID).
		Count(&count)
	if res.Error != nil {
		return res.Error
	}
	if count > 0 {
		res = db.conn.Where("path = ?", obj.Path).Save(obj)
	} else {
		res = db.conn.Create(obj)
	}
	return res.Error
}

// Delete file base object
func (db *connector) Delete(path string) error {
	return db.conn.Where("path = ?", path).
		Delete((*models.Object)(nil)).Error
}
