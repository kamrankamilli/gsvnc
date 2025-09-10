package util

import (
	"encoding/binary"
	"errors"
	"io"
	"reflect"
)

// PackStruct writes struct fields to buf in declaration order (BigEndian).
func PackStruct(buf io.Writer, data interface{}) error {
	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("Data is invalid (nil or non-pointer)")
	}
	val := rv.Elem()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		v := field.Interface()
		// Strings should be pre-sized elsewhere (not common in RFB structs)
		if err := binary.Write(buf, binary.BigEndian, v); err != nil {
			return err
		}
	}
	return nil
}

// Write writes any value in BigEndian to buf.
func Write(buf io.Writer, v interface{}) error {
	return binary.Write(buf, binary.BigEndian, v)
}
