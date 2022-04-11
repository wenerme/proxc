package dbcache

import (
	"net/http"

	"github.com/wenerme/proxc/httpcache/dbcache/models"
	"gorm.io/gorm"
)

type Cache struct {
	GetDB func(r *http.Request) (*gorm.DB, *gorm.DB, error)
}

func (d *Cache) SetResponse(resp *http.Response) (err error) {
	db, file, err := d.GetDB(resp.Request)
	if err != nil {
		return err
	}
	return SetResponse(&SetResponseOptions{
		DB:       db,
		FileDB:   file,
		Response: resp,
	})
}

func (d *Cache) GetResponse(req *http.Request) (resp *http.Response, err error) {
	db, file, err := d.GetDB(req)
	if err != nil {
		return
	}
	return GetResponse(&GetResponseOptions{
		DB:      db,
		FileDB:  file,
		Request: req,
	})
}

func (d *Cache) DeleteResponse(req *http.Request) error {
	db, _, err := d.GetDB(req)
	if err != nil {
		return err
	}
	// delete file ?
	out := models.HTTPResponse{}
	return db.Where(models.HTTPResponse{
		Method: req.Method,
		URL:    req.URL.String(),
	}).Delete(&out).Error
}
