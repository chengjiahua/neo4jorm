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
	}
	return result
}

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

		props[propName] = rv.Field(i).Interface()
	}
	return props, nil
}

func decodeNode(node neo4j.Node, out interface{}) error {
	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("%w: output must be a non-nil pointer")
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