package g

import (
	"reflect"
	"strings"
)

// StructField is a field of a struct
type StructField struct {
	Field reflect.StructField
	Value reflect.Value
	JKey  string
}

// IsPointer returns `true` is obj is a pointer
func IsPointer(obj interface{}) bool {
	value := reflect.ValueOf(obj)
	return value.Kind() == reflect.Ptr
}

// IsSlice returns `true` is obj is a slice
func IsSlice(obj interface{}) bool {
	value := reflect.ValueOf(obj)
	return value.Kind() == reflect.Slice || value.Kind() == reflect.Array
}

// StructFieldsMapToKey returns a map of fields name to key
func StructFieldsMapToKey(obj interface{}) (m map[string]string) {
	m = map[string]string{}
	for _, f := range StructFields(obj) {
		m[f.Field.Name] = f.JKey
		m[f.JKey] = f.JKey
	}
	return
}

// StructFields returns the fields of a struct
func StructFields(obj interface{}) (fields []StructField) {
	var t reflect.Type
	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		t = reflect.Indirect(value).Type()
	} else {
		t = reflect.TypeOf(obj)
	}
	for i := 0; i < t.NumField(); i++ {
		var valueField reflect.Value
		if value.Kind() == reflect.Ptr {
			valueField = value.Elem().Field(i)
		} else {
			valueField = value.Field(i)
		}
		sField := t.Field(i)
		jKey := strings.Split(sField.Tag.Get("json"), ",")[0]
		fields = append(fields, StructField{sField, valueField, jKey})
	}
	return fields
}

// CloneValue clones a pointer to another
func CloneValue(source interface{}, destin interface{}) {
	x := reflect.ValueOf(source)
	if x.Kind() == reflect.Ptr {
		starX := x.Elem()
		y := reflect.New(starX.Type())
		starY := y.Elem()
		starY.Set(starX)
		reflect.ValueOf(destin).Elem().Set(y.Elem())
	} else {
		destin = x.Interface()
	}
}
