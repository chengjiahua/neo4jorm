package neo4jorm

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type Model struct {
	client      *Client
	modelType   reflect.Type
	labels      []string
	primaryKey  string
	fieldMap    map[string]string // 结构体字段到属性的映射
	generated   map[string]bool   // 生成的字段
}

func newModel(client *Client, model interface{}) *Model {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	m := &Model{
		client:    client,
		modelType: modelType,
		fieldMap:  make(map[string]string),
		generated: make(map[string]bool),
	}

	m.parseTags()
	return m
}

func (m *Model) parseTags() {
	for i := 0; i < m.modelType.NumField(); i++ {
		field := m.modelType.Field(i)
		tag := field.Tag.Get(tagName)
		if tag == "" {
			continue
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
		if name, ok := tags["name"]; ok {
			propName = name
		}
		m.fieldMap[field.Name] = propName
	}
}

// 批量创建节点
func (m *Model) CreateBatch(nodes []interface{}) error {
	session := m.client.driver.NewSession(neo4j.SessionConfig{
		DatabaseName: m.client.config.Database,
	})
	defer session.Close()

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		query, params := buildCreateBatchQuery(m, nodes)
		result, err := tx.Run(query, params)
		if err != nil {
			return nil, fmt.Errorf("session WriteTransaction run error: %v",  err)
		}
		return result.Consume()
	}, neo4j.WithTxTimeout(30*time.Second))

	return err
}

func buildCreateBatchQuery(m *Model, nodes []interface{}) (string, map[string]interface{}) {
	var sb strings.Builder
	sb.WriteString("UNWIND $nodes AS node\n")
	sb.WriteString("CREATE (n")
	
	// 添加标签
	for _, label := range m.labels {
		sb.WriteString(":" + label)
	}
	sb.WriteString(")\n")
	
	// 设置属性
	sb.WriteString("SET n += node.props\n")
	
	// 处理生成的主键
	if m.primaryKey != "" && m.generated[m.primaryKey] {
		sb.WriteString(fmt.Sprintf("SET n.%s = apoc.create.uuid()\n", m.fieldMap[m.primaryKey]))
	}

	// 准备参数
	processed := make([]map[string]interface{}, 0, len(nodes))
	for _, node := range nodes {
		props, _ := structToProperties(node)
		processed = append(processed, map[string]interface{}{"props": props})
	}

	return sb.String(), map[string]interface{}{"nodes": processed}
}

// 更新节点
func (m *Model) Update(node interface{}) error {
	props, err := structToProperties(node)
	if err != nil {
		return err
	}

	pkValue := reflect.ValueOf(node).Elem().FieldByName(m.primaryKey).Interface()
	query := fmt.Sprintf(
		"MATCH (n:%s {%s: $pk}) SET n += $props",
		strings.Join(m.labels, ":"), 
		m.fieldMap[m.primaryKey],
	)

	session := m.client.driver.NewSession(neo4j.SessionConfig{
		DatabaseName: m.client.config.Database,
	})
	defer session.Close()

	_, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(query, map[string]interface{}{
			"pk":    pkValue,
			"props": props,
		})
		if err != nil {
			return nil, err
		}
		return result.Consume()
	})
	return err
}