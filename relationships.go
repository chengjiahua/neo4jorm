package neo4jorm

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

// RelationshipConfig 存储关系配置
type RelationshipConfig struct {
	Type      string
	Direction string // incoming, outgoing, both
	Merge     bool
}

type Relationship struct {
	From      interface{}
	To        interface{}
	Type      string
	Props     map[string]interface{}
	Direction string
}

// createRelationship 创建关系对象
func createRelationship(from, to interface{}, config RelationshipConfig) Relationship {
	return Relationship{
		From:      from,
		To:        to,
		Type:      config.Type,
		Direction: config.Direction,
		Props:     extractRelationshipProps(to),
	}
}

// extractRelationshipProps 提取关系属性
func extractRelationshipProps(obj interface{}) map[string]interface{} {
	props := make(map[string]interface{})
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	// 提取嵌套属性
	if rv.Kind() == reflect.Struct {
		for i := 0; i < rv.NumField(); i++ {
			field := rv.Type().Field(i)
			if tag := field.Tag.Get(tagName); tag != "" {
				props[strings.Split(tag, ",")[0]] = rv.Field(i).Interface()
			}
		}
	}
	return props
}

// CreateRelation 创建关系（使用主键判断）
// Relation 定义关系输入结构
type Relation struct {
	Start interface{} // 起始节点结构体
	End   interface{} // 目标节点结构体
}

// 批量创建无属性关系方法
func (m *Model) CreateRelations(relations []Relation, relType string) error {
	// 空关系列表直接返回，避免无效操作
	if len(relations) == 0 {
		return nil
	}

	session := m.client.driver.NewSession(neo4j.SessionConfig{
		DatabaseName: m.client.config.Database,
	})
	defer session.Close()

	// 使用UNWIND优化批量操作
	var query strings.Builder
	query.WriteString("UNWIND $rels AS rel ")
	query.WriteString("MERGE (a:%s { %s: rel.startVal }) ")
	query.WriteString("MERGE (b:%s { %s: rel.endVal }) ")
	query.WriteString("MERGE (a)-[r1:%s]->(b) ")
	query.WriteString("MERGE (a)-[r2:%s]->(b) ")
	var (
		startPK string
		endPK   string
		start   *Model
		end     *Model
	)

	// 提前校验并获取元数据
	if len(relations) > 0 {
		// 获取第一个关系的元数据（假设所有关系类型相同）
		firstRel := relations[0]
		start = newModel(m.client, firstRel.Start)
		end = newModel(m.client, firstRel.End)
		startPK = start.fieldMap[start.primaryKey]
		endPK = end.fieldMap[end.primaryKey]
	}

	// 构建最终查询
	finalQuery := fmt.Sprintf(query.String(),
		start.table, startPK,
		end.table, endPK,
		relType,
		relType,
	)

	// 准备批量参数
	relsParams := make([]map[string]interface{}, 0, len(relations))
	for _, rel := range relations {

		relsParams = append(relsParams, map[string]interface{}{
			"startVal": getStructKeyValue(rel.Start, start.primaryKey),
			"endVal":   getStructKeyValue(rel.End, end.primaryKey),
		})
	}

	params := map[string]interface{}{
		"rels": relsParams,
	}

	if m.debug {
		fmt.Printf("Executing CreateRelations:\n%s\nWith params: %+v\n", finalQuery, params)
	}

	// 执行批量操作
	_, err := session.Run(finalQuery, params)
	return err
}

// DeleteRelation 删除关系（使用主键判断）
func (m *Model) DeleteRelations(relations []Relation, relType string) error {
	// 空关系列表直接返回，避免无效操作
	if len(relations) == 0 {
		return nil
	}

	session := m.client.driver.NewSession(neo4j.SessionConfig{
		DatabaseName: m.client.config.Database,
	})
	defer session.Close()

	var query strings.Builder
	// 使用UNWIND批量处理，MATCH定位关系后删除
	query.WriteString("UNWIND $rels AS rel ")
	query.WriteString("MATCH (a:%s {%s: rel.startVal})-[r:%s]->(b:%s {%s: rel.endVal}) ")
	query.WriteString("DELETE r")

	// 获取元数据（复用原有逻辑）
	firstRel := relations[0]
	start := newModel(m.client, firstRel.Start)
	end := newModel(m.client, firstRel.End)
	startPK := start.fieldMap[start.primaryKey]
	endPK := end.fieldMap[end.primaryKey]

	// 构建最终查询
	finalQuery := fmt.Sprintf(query.String(),
		start.table, startPK,
		relType,
		end.table, endPK,
	)

	// 准备批量参数（与CreateRelations保持相同结构）
	relsParams := make([]map[string]interface{}, 0, len(relations))
	for _, rel := range relations {
		relsParams = append(relsParams, map[string]interface{}{
			"startVal": getStructKeyValue(rel.Start, start.primaryKey),
			"endVal":   getStructKeyValue(rel.End, end.primaryKey),
		})
	}

	params := map[string]interface{}{
		"rels": relsParams,
	}

	if m.debug {
		fmt.Printf("Executing Delete:\n%s\nWith params: %+v\n", finalQuery, params)
	}

	// 执行删除操作
	_, err := session.Run(finalQuery, params)
	return err
}

// 获取主键值的辅助函数
func getStructKeyValue(model interface{}, fieldKey string) (Value interface{}) {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	return val.FieldByName(fieldKey).Interface()
}
