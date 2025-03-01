package neo4jorm

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

const (
	tagName      = "neo4j"
	tagPrimary   = "primary"
	tagLabel     = "label"
	tagGenerated = "generated"
	tagkey       = "name"
)

func parseTag(tag string) map[string]string {
	result := make(map[string]string)
	parts := strings.Split(tag, ";")
	for _, part := range parts {
		kv := strings.Split(part, ":")
		if len(kv) >= 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(strings.Join(kv[1:], ":"))
			result[key] = value
		}

		if len(kv) == 1 && kv[0] == tagPrimary {
			key := strings.TrimSpace(kv[0])
			result[key] = ""
		}
	}
	return result
}

func decodeNode(node neo4j.Node, out interface{}) error {
	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("decodeNode: output must be a non-nil pointer")
	}

	rv = rv.Elem()
	rt := rv.Type()

	props := node.Props
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag.Get(tagName)
		if tag == "" {
			continue
		}

		tags := parseTag(tag)
		propName := field.Name
		if name, ok := tags["name"]; ok {
			propName = name
		}

		if value, ok := props[propName]; ok {
			fieldValue := rv.Field(i)
			val := reflect.ValueOf(value)
			if val.Type().ConvertibleTo(fieldValue.Type()) {
				fieldValue.Set(val.Convert(fieldValue.Type()))
			}
		}
	}
	return nil
}

// structToProperties 将结构体转换为属性
func structToProperties(v interface{}) (map[string]interface{}, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("structToProperties: expected struct, got %T", v)
	}

	props := make(map[string]interface{})
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag.Get(tagName)
		if tag == "" {
			continue
		}

		tags := parseTag(tag)
		if _, ok := tags[tagGenerated]; ok {
			continue
		}

		propName := field.Name
		if name, ok := tags["name"]; ok {
			propName = name
		}

		fieldValue := rv.Field(i)
		if !isZeroValue(fieldValue) {
			props[propName] = fieldValue.Interface()
		}

	}
	return props, nil
}

// isZeroValue 判断是否为零值
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	default:
		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	}
}
