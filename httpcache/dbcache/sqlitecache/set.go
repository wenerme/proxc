package sqlitecache

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	_ "github.com/glebarez/go-sqlite" //nolint:revive
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type (
	GetDBFunc    func(key string, opts ...func(*GetDBOptions)) (db *gorm.DB, err error)
	GetDBOptions struct {
		OnInit func(db *gorm.DB) error
		Pragma map[string]interface{}
		Params map[string]string
		File   string
		Config *gorm.Config
	}
)

type Set struct {
	DBs map[string]*gorm.DB
	Dir string
	l   sync.RWMutex
}

func (d *Set) Get(key string, opts ...func(*GetDBOptions)) (db *gorm.DB, err error) {
	d.l.Lock()
	defer d.l.Unlock()

	if d.DBs == nil {
		d.DBs = make(map[string]*gorm.DB)
	}
	if db = d.DBs[key]; db != nil {
		return
	}

	o := &GetDBOptions{
		OnInit: func(db *gorm.DB) error {
			return nil
		},
		Pragma: map[string]interface{}{
			// "page_size":          65536,
			// "incremental_vacuum": 1000, // 8MB

			"synchronous":        0,
			"journal_mode":       "WAL",
			"page_size":          8192, // 8K
			"auto_vacuum":        0,
			"busy_timeout":       3000, // 3s
			"wal_autocheckpoint": 2000, // 16MB
		},
		Params: map[string]string{},
		Config: &gorm.Config{},
	}

	var dir string
	dir, err = filepath.Abs(d.Dir)
	if err != nil {
		return
	}
	o.File = filepath.Join(dir, key+".sqlite")
	for _, opt := range opts {
		opt(o)
	}

	dsn := &url.URL{
		Scheme: "file",
		Path:   o.File,
	}

	for k, v := range o.Pragma {
		dsn.RawQuery += "&_pragma=" + k + "(" + fmt.Sprint(v) + ")"
	}
	for k, v := range o.Params {
		dsn.RawQuery += "&" + k + "=" + v
	}
	dsn.RawQuery = strings.TrimPrefix(dsn.RawQuery, "&")

	db, err = gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        dsn.String(),
		// Conn: db,
	}, o.Config)
	if err == nil {
		err = o.OnInit(db)
	}
	if err == nil {
		d.DBs[key] = db
	}
	return
}
