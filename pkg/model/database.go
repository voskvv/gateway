package model

import (
	"time"

	"github.com/fatih/structs"
)

//DatabaseConfig containts DB config
type DatabaseConfig struct {
	User          string
	Password      string
	Database      string
	Address       string
	Port          string
	SafeMigration bool
	Debug         bool
}

const timeFormat = "Monday, 02 Jan 2006 15:04:05 MST"

//DefaultModel embedded struct in each Gorm Model
type DefaultModel struct {
	ID        string     `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" structs:"id"`
	CreatedAt time.Time  `structs:"-"`
	UpdatedAt time.Time  `structs:"-"`
	DeletedAt *time.Time `structs:"-"`
}

func (dm DefaultModel) getMap() map[string]interface{} {
	model := structs.Map(dm)
	model["created_at"] = dm.CreatedAt.Format(timeFormat)
	model["updated_at"] = dm.UpdatedAt.Format(timeFormat)
	if dm.DeletedAt != nil {
		model["deleted_at"] = dm.DeletedAt.Format(timeFormat)
	} else {
		model["deleted_at"] = "null"
	}
	return model
}
