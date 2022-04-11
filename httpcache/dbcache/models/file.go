package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

type FileContent struct {
	Model
	Hash        string `gorm:"unique"`
	Name        string
	Size        int64 `gorm:"index"`
	Ext         string
	ContentType string
	Content     []byte
	Extension   datatypes.JSON
	Attributes  datatypes.JSON
}

func (FileContent) ConflictColumns() []clause.Column {
	return []clause.Column{{Name: "hash"}}
}

type FileRef struct {
	Model
	Hash string       `gorm:"uniqueIndex:idx_file_ref_hash_url"`
	URL  string       `gorm:"uniqueIndex:idx_file_ref_hash_url"`
	File *FileContent `gorm:"foreignKey:Hash;references:Hash"`
	Name string
}

func (FileRef) ConflictColumns() []clause.Column {
	return []clause.Column{{Name: "hash"}, {Name: "url"}}
}
