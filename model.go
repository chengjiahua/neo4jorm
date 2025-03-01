package neo4jorm

import (
	"fmt"
	"reflect"
)

type Model struct {
	debug      bool
	client     *Client
	modelType  reflect.Type
	elemType   reflect.Type // 新增字段，保存切片元素类型
	labels     []string
	primaryKey string
	fieldMap   map[string]string
	generated  map[string]bool
}

// 修改model.go中的newModel函数
func newModel(client *Client, model interface{}) *Model {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	m := &Model{
		client:    client,
		debug:     client.debug,
		modelType: modelType,
		elemType:  reflect.PtrTo(modelType), // 保存切片元素类型
		fieldMap:  make(map[string]string),
		generated: make(map[string]bool),
	}

	m.parseTags()
	if m.debug {
		m.DebugInfo()
	}
	return m
}

func (m *Model) setDebug(debug bool) {
	m.debug = debug
}

func (m *Model) parseTags() {
	for i := 0; i < m.modelType.NumField(); i++ {
		field := m.modelType.Field(i)
		tag := field.Tag.Get(tagName)
		if tag == "" {
			tag = field.Name
		}

		tags := parseTag(tag)
		// 处理标签
		if label, ok := tags[tagLabel]; ok {
			m.labels = append(m.labels, label)
		}
		if _, ok := tags[tagPrimary]; ok {
			m.primaryKey = field.Name
		}
		if _, ok := tags[tagGenerated]; ok {
			m.generated[field.Name] = true
		}

		// 处理属性名称映射
		propName := field.Name
		if name, ok := tags[tagkey]; ok {
			propName = name
		}
		m.fieldMap[field.Name] = propName
	}
}

// 在model.go中添加调试方法
func (m *Model) DebugInfo() *Model {
	m.setDebug(true)
	fmt.Println(fmt.Sprintf(
		"Model{"+
			" modelType:%s"+
			" elemType:%s"+
			" labels:%v"+
			" primaryKey:%s"+
			" fieldMap:%v"+
			" generated:%v"+
			"}",
		m.modelType.String(),
		m.elemType.String(),
		m.labels,
		m.primaryKey,
		m.fieldMap,
		m.generated))
	return m
}
