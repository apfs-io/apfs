package gorm

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/demdxx/gocast/v2"
	"gorm.io/gorm"

	"github.com/apfs-io/apfs/internal/storage"
	"github.com/apfs-io/apfs/models"
)

type openFnk func(dsn string) gorm.Dialector

var dialectors = map[string]openFnk{}

// Connector object
type connector struct {
	conn *gorm.DB
}

// Connect database by URL
func Connect(ctx context.Context, connectURL string) (storage.DB, error) {
	u, err := url.Parse(connectURL)
	if err != nil {
		return nil, err
	}
	return New(ctx, connectURL,
		gocast.Bool(u.Query().Get(`automigrate`)),
		gocast.Bool(u.Query().Get(`debug`)))
}

// New db connector
func New(ctx context.Context, connect string, automigrate, debug bool) (*connector, error) {
	conn, err := connectDB(ctx, connect, debug)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(connect, "mysql") {
		conn = conn.Set("gorm:table_options", "ENGINE=InnoDB")
	}
	if automigrate {
		if err := conn.AutoMigrate(&models.Object{}); err != nil {
			return nil, err
		}
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
		count int64
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

// Close database connection
func (db *connector) Close() error {
	return nil
}

// Connect to database
func connectDB(ctx context.Context, connection string, debug bool) (*gorm.DB, error) {
	var (
		i      = strings.Index(connection, "://")
		driver = connection[:i]
	)
	if driver == "mysql" {
		connection = connection[i+3:]
	}
	openDriver := dialectors[driver]
	if openDriver == nil {
		return nil, fmt.Errorf(`unsupported database driver %s`, driver)
	}
	db, err := gorm.Open(openDriver(connection), &gorm.Config{SkipDefaultTransaction: true})
	if err == nil && debug {
		db = db.Debug()
	}
	return db.WithContext(ctx), err
}
