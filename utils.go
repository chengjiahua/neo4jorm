package neo4jorm

import (
	"fmt"
	"math"
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

// convertToInt 尝试将值转换为int64
func convertToInt(val reflect.Value) (int64, bool) {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int(), true
	case reflect.Float32, reflect.Float64:
		// 处理浮点型截断
		return int64(val.Float()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// 处理无符号整型
		return int64(val.Uint()), true
	case reflect.Bool:
		// 处理布尔型（true=1，false=0）
		if val.Bool() {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}

// convertToUint 尝试将值转换为uint64
func convertToUint(val reflect.Value) (uint64, bool) {
	switch val.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// 处理负数情况
		if ival := val.Int(); ival >= 0 {
			return uint64(ival), true
		}
	case reflect.Float32, reflect.Float64:
		// 处理浮点范围和精度
		if fval := val.Float(); fval >= 0 && fval <= math.MaxUint64 {
			return uint64(fval), true
		}
	case reflect.Bool:
		if val.Bool() {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// convertToFloat 尝试将值转换为float64
func convertToFloat(val reflect.Value) (float64, bool) {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(val.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(val.Uint()), true
	case reflect.Float32, reflect.Float64:
		return val.Float(), true
	case reflect.Bool:
		if val.Bool() {
			return 1.0, true
		}
		return 0.0, true
	default:
		return 0, false
	}
}
