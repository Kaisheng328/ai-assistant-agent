package model

type Setting struct {
	Key   string `gorm:"primaryKey;size:255" json:"key"`
	Value string `gorm:"type:text;not null" json:"value"`
}
