package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type OriginS3Config map[string]string
type StringArray []string

type Sites struct {
	ID           uint        `json:"id" gorm:"primaryKey"`
	Title        string      `json:"title"`
	Domain       string      `json:"domain"`
	AllowedTypes StringArray `json:"allowed_type"`
	Origin       Origin      `json:"origin"`
}

type Origin struct {
	URL      string         `json:"url"`
	S3Config OriginS3Config `json:"s3_config"`
}



// Scan implements the sql.Scanner interface and Value implements the driver.Valuer interface
func (sa *StringArray) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, sa)
}

func (sa StringArray) Value() (driver.Value, error) {
	return json.Marshal(sa)
}

func (osc *OriginS3Config) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, osc)
}

func (osc OriginS3Config) Value() (driver.Value, error) {
	return json.Marshal(osc)
}

func (o *Origin) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, o)
}

func (o Origin) Value() (driver.Value, error) {
	return json.Marshal(o)
}
