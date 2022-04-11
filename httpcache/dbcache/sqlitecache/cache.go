package sqlitecache

import (
	"net/http"

	"github.com/wenerme/proxc/httpcache/dbcache"
	"github.com/wenerme/proxc/httpcache/dbcache/models"
	"gorm.io/gorm"
)

// NewSQLiteCache create a cache for per host sqlite db plus a file db
func NewSQLiteCache(dir string) *dbcache.Cache {
	set := &Set{
		Dir: dir,
	}
	return &dbcache.Cache{
		GetDB: func(r *http.Request) (db *gorm.DB, file *gorm.DB, err error) {
			db, err = set.Get(r.URL.Host, func(o *GetDBOptions) {
				o.OnInit = func(db *gorm.DB) error {
					return db.AutoMigrate(models.HTTPResponse{})
				}
			})
			if err != nil {
				return
			}
			file, err = set.Get("file", func(o *GetDBOptions) {
				o.OnInit = func(db *gorm.DB) error {
					return db.AutoMigrate(models.FileContent{}, models.FileRef{})
				}
			})
			return
		},
	}
}

// NewMemoryCache create a cache use memory sqlite
func NewMemoryCache() *dbcache.Cache {
	set := &Set{}
	return &dbcache.Cache{
		GetDB: func(r *http.Request) (*gorm.DB, *gorm.DB, error) {
			db, err := set.Get("mem", func(opts *GetDBOptions) {
				opts.Params["mode"] = "memory"
				opts.OnInit = func(db *gorm.DB) error {
					return db.AutoMigrate(models.HTTPResponse{}, models.FileContent{}, models.FileRef{})
				}
			})
			return db, db, err
		},
	}
}
