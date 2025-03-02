package neo4jorm

import (
	"fmt"
	"reflect"
	"sync"
)

// 全局模型注册器
var modelRegistry = &Registry{
	models: &sync.Map{},
}

type Registry struct {
	models *sync.Map // 存储 [reflect.Type]*Model
}

type Model struct {
	debug      bool
	client     *Client
	modelType  reflect.Type
	elemType   reflect.Type // 新增字段，保存切片元素类型
	table      string
	primaryKey string
	fieldMap   map[string]string
	generated  map[string]bool

	//查询参数
	conditions []string               // 存储WHERE条件表达式
	params     map[string]interface{} // 查询参数
	orderBy    []string               // 排序字段
	limit      int                    // 限制结果数量
}

func (m *Model) register() error {
	modelRegistry.models.Store(m.modelType, m)
	return nil
}

// 获取已注册模型
func getModel(obj interface{}) (*Model, bool) {
	t := getType(obj)
	val, ok := modelRegistry.models.Load(t)
	if !ok {
		return nil, false
	}
	return val.(*Model).clone(), true
}

// 修改model.go中的newModel函数
func newModel(client *Client, model interface{}) *Model {
	m, ok := getModel(model)
	if ok {
		return m
	}

	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	m = &Model{
		client:    client,
		debug:     client.debug,
		modelType: modelType,
		elemType:  modelType, // 保存切片元素类型
		fieldMap:  make(map[string]string),
		generated: make(map[string]bool),
	}

	m.parseTags()
	m.register()

	if m.debug {
		m.DebugInfo()
	}
	return m
}

func (m *Model) clone() *Model {
	return &Model{
		debug:      m.debug,
		client:     m.client,
		modelType:  m.modelType,
		elemType:   m.elemType,
		table:      m.table,
		primaryKey: m.primaryKey,
		fieldMap:   m.fieldMap,
		generated:  m.generated,
		conditions: m.conditions,
		params:     m.params,
		orderBy:    m.orderBy,
		limit:      m.limit,
	}
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
		if table, ok := tags[tagTable]; ok {
			m.table = table
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
			" table:%v"+
			" primaryKey:%s"+
			" fieldMap:%v"+
			" generated:%v"+
			" conditions:%v"+
			" params:%v"+
			" orderBy:%v"+
			" limit:%v"+
			"}",
		m.modelType.String(),
		m.elemType.String(),
		m.table,
		m.primaryKey,
		m.fieldMap,
		m.generated,
		m.conditions,
		m.params,
		m.orderBy,
		m.limit,
	))
	return m
}
