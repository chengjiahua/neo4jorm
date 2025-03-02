package neo4jorm

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type RelationshipConfig struct {
	Type      string // 关系类型
	Direction string // 方向: incoming/outgoing
	Merge     bool   // 是否使用MERGE
}

type Relationship struct {
	From      interface{}
	To        interface{}
	Type      string
	Direction string
	Props     map[string]interface{}
}

func (c *Client) Relate(rel *Relationship) error {
	session := c.driver.NewSession(neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	defer session.Close()

	// 获取模型信息
	fromModel := c.Model(rel.From)
	toModel := c.Model(rel.To)

	// 构建Cypher
	query, params := buildRelationshipQuery(fromModel, toModel, rel)

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(query, params)
		if err != nil {
			return nil, err
		}
		return result.Consume()
	})
	return err
}

func buildRelationshipQuery(from, to *Model, rel *Relationship) (string, map[string]interface{}) {
	var sb strings.Builder
	params := make(map[string]interface{})

	// MATCH起始节点
	sb.WriteString(fmt.Sprintf("MATCH (a:%s {%s: $fromPk})\n",
		strings.Join(from.labels, ":"),
		from.fieldMap[from.primaryKey]))
	params["fromPk"] = getPrimaryKeyValue(rel.From)

	// MATCH目标节点
	sb.WriteString(fmt.Sprintf("MATCH (b:%s {%s: $toPk})\n",
		strings.Join(to.labels, ":"),
		to.fieldMap[to.primaryKey]))
	params["toPk"] = getPrimaryKeyValue(rel.To)

	// 创建关系
	direction := "-"
	switch rel.Direction {
	case "LEFT":
		direction = "<-"
	case "RIGHT":
		direction = "->"
	default:
		direction = "-"
	}

	sb.WriteString(fmt.Sprintf("CREATE (a)%s[:%s $props]%s(b)\n",
		direction[:1], // 第一个方向符号
		rel.Type,
		direction[1:])) // 第二个方向符号

	// 添加属性
	if rel.Props != nil {
		props, _ := structToProperties(rel.Props)
		params["props"] = props
	} else {
		params["props"] = map[string]interface{}{}
	}

	return sb.String(), params
}

func getPrimaryKeyValue(node interface{}) interface{} {
	v := reflect.ValueOf(node)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	model := Model{} // 这里实际应该通过反射获取主键字段
	field := v.FieldByName(model.primaryKey)
	return field.Interface()
}


