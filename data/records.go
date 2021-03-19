package data

import (
	"strings"

	"github.com/spf13/cast"
)

// Fields are field names
type Fields []string

// Row is one record / row
type Row []interface{}

// Dataset is a collection of rows / records
type Dataset struct {
	Fields   Fields
	FieldMap map[string]int
	Rows     []Row
}

// NewDataset creates a new dataset
func NewDataset(fields ...string) Dataset {
	return Dataset{
		Fields:   fields,
		FieldMap: Fields(fields).AsMap(),
		Rows:     []Row{},
	}
}

func (d *Dataset) String(i int, key string) string {
	col, ok := d.FieldMap[strings.ToLower(key)]
	if !ok {
		return ""
	}
	return cast.ToString(d.Rows[i][col])
}

// AsMap returns a mpa of index
func (fs Fields) AsMap() map[string]int {
	fm := map[string]int{}
	for i, f := range fs {
		fm[strings.ToLower(f)] = i
	}
	return fm
}
