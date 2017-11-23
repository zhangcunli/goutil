package util

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math/rand"
	"reflect"
	"time"
)

func RandStagger(t time.Duration) time.Duration {
	return time.Duration(uint64(rand.Int63()) % uint64(t))
}

func writeBuf(w io.Writer, v reflect.Value) (n int, err error) {
	newBuf := bytes.NewBuffer(nil)
	for i := 0; i < v.NumField(); i++ {
		switch v.Field(i).Type().Kind() {
		case reflect.Struct:
			n, err := writeBuf(newBuf, v.Field(i))
			if err != nil {
				return n, err
			}
		case reflect.Bool:
			boolByte := []byte{0}
			if v.Field(i).Bool() {
				boolByte = []byte{1}
			}
			newBuf.Write(boolByte)
		case reflect.String:
			newBuf.WriteString(v.Field(i).String())
		case reflect.Slice:
			newBuf.Write(v.Field(i).Bytes())
		case reflect.Int:
			binary.Write(newBuf, binary.BigEndian, int32(v.Field(i).Int()))
		case reflect.Uint:
			binary.Write(newBuf, binary.BigEndian, uint32(v.Field(i).Uint()))
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
			binary.Write(newBuf, binary.BigEndian, v.Field(i).Interface())
		}
	}
	return w.Write(newBuf.Bytes())
}

func WriteStructToBuffer(w io.Writer, data interface{}) error {
	v := reflect.Indirect(reflect.ValueOf(data))
	if v.Kind() == reflect.Struct {
		_, err := writeBuf(w, v)
		return err
	}
	return errors.New("invalid type Not a struct")
}
