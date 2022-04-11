package dbcache

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/wenerme/proxc/httpcache/dbcache/models"
	"go.uber.org/multierr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SetResponseOptions struct {
	DB                  *gorm.DB
	FileDB              *gorm.DB
	Response            *http.Response
	Dry                 bool
	Responses           []*models.HTTPResponse
	FileContents        []*models.FileContent
	FileRefs            []*models.FileRef
	OnConflictDoNothing bool
}
type GetResponseOptions struct {
	DB      *gorm.DB
	FileDB  *gorm.DB
	Request *http.Request
}

func GetResponse(o *GetResponseOptions) (resp *http.Response, err error) {
	if o.FileDB == nil {
		o.FileDB = o.DB
	}
	var out models.HTTPResponse
	req := o.Request
	err = o.DB.Where(models.HTTPResponse{
		Method: req.Method,
		URL:    req.URL.String(),
	}).Limit(1).Find(&out).Error
	if err != nil || out.URL == "" {
		return
	}
	resp, err = out.GetResponse(req)
	if err != nil {
		return
	}
	if out.ContentHash != "" {
		var file models.FileContent
		err = o.FileDB.Where(models.FileContent{
			Hash: out.ContentHash,
		}).Limit(1).Find(&file).Error
		if err != nil {
			return
		}
		if file.Content == nil {
			log.Error().Str("hash", out.ContentHash).Msgf("file not found")
		}
		resp.Body = io.NopCloser(bytes.NewReader(file.Content))
		resp.Header.Set("Content-Hash", file.Hash)
	}
	return
}

var DetectExt = func(name string, data []byte) string {
	return filepath.Ext(name)
}

func SetResponse(o *SetResponseOptions) (err error) {
	if o.FileDB == nil {
		o.FileDB = o.DB
	}
	hr := &models.HTTPResponse{}
	err = hr.SetResponse(o.Response)
	if err != nil {
		return
	}
	if hr.FileName != "" && hr.BodySize > 0 {
		fc := &models.FileContent{
			Hash:        hr.ContentHash,
			Name:        hr.FileName,
			Size:        hr.BodySize,
			ContentType: hr.ContentType,
			Content:     hr.Body,
		}
		fc.Ext = DetectExt(fc.Name, fc.Content)
		ref := &models.FileRef{
			Hash: fc.Hash,
			Name: fc.Name,
			URL:  hr.URL,
		}
		if !o.Dry {
			err = multierr.Combine(
				o.FileDB.Clauses(clause.OnConflict{Columns: fc.ConflictColumns(), DoNothing: true}).Create(fc).Error,
				o.FileDB.Clauses(clause.OnConflict{Columns: ref.ConflictColumns(), DoNothing: true}).Create(ref).Error,
			)
		} else {
			o.FileContents = append(o.FileContents, fc)
			o.FileRefs = append(o.FileRefs, ref)
		}

		if err != nil {
			return
		}
		hr.Body = nil
		hr.BodySize = 0
	}
	if !o.Dry {
		conflict := clause.OnConflict{Columns: hr.ConflictColumns(), DoNothing: o.OnConflictDoNothing}
		if !conflict.DoNothing {
			conflict.UpdateAll = true
		}
		err = o.DB.Clauses(conflict).Create(hr).Error
	} else {
		o.Responses = append(o.Responses, hr)
	}

	return
}
