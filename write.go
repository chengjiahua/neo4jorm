package neo4jorm

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

func (m *Model) CreateOne(node interface{}) error {
	// 将单个节点包装成切片调用MergeBatch
	nodes := []interface{}{node}
	return m.CreateBatch(nodes)
}

// 批量创建节点，需自行创建唯一约束
func (m *Model) CreateBatch(nodes interface{}) error {
	// 添加类型验证
	nodesValue := reflect.ValueOf(nodes)
	if nodesValue.Kind() != reflect.Slice && nodesValue.Kind() != reflect.Array {
		return fmt.Errorf("%s: expected slice/array, got %T", ErrInvalidModel, nodes)
	}

	session := m.client.driver.NewSession(neo4j.SessionConfig{
		DatabaseName: m.client.config.Database,
	})
	defer session.Close()

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(buildCreateBatchQuery(m, nodesValue))
		if err != nil {
			return nil, fmt.Errorf("create batch failed: %w", err)
		}
		return result.Consume()
	}, neo4j.WithTxTimeout(30*time.Second))

	return err
}

func buildCreateBatchQuery(m *Model, nodesValue reflect.Value) (string, map[string]interface{}) {
	var sb strings.Builder
	sb.WriteString("UNWIND $nodes AS node ")
	sb.WriteString("CREATE (n")
	sb.WriteString(":" + m.table)
	sb.WriteString(") ")

	// 设置属性
	sb.WriteString("SET n += node.props ")

	// 处理生成的主键
	if m.generated[m.primaryKey] {
		sb.WriteString(fmt.Sprintf("SET n.%s = coalesce(n.%s, apoc.create.uuid()) ",
			m.fieldMap[m.primaryKey],
			m.fieldMap[m.primaryKey]))
	}

	// 准备参数
	processed := make([]map[string]interface{}, 0, nodesValue.Len())
	for i := 0; i < nodesValue.Len(); i++ {
		node := nodesValue.Index(i).Interface()
		props, _ := structToProperties(node)
		processed = append(processed, map[string]interface{}{"props": props})
	}
	params := map[string]interface{}{"nodes": processed}
	if m.debug {
		fmt.Println(sb.String(), params)
	}
	return sb.String(), params
}

// 更新节点
func (m *Model) Update(node interface{}) error {
	props, err := structToProperties(node)
	if err != nil {
		return err
	}

	pkValue := props[m.fieldMap[m.primaryKey]]
	query := fmt.Sprintf(
		"MATCH (n:%s {%s: $pk}) SET n += $props",
		m.table,
		m.fieldMap[m.primaryKey],
	)

	params := map[string]interface{}{
		"pk":    pkValue,
		"props": props,
	}

	if m.debug {
		fmt.Printf("Executing Update:\n%s\nWith params: %+v\n", query, params)
	}

	session := m.client.driver.NewSession(neo4j.SessionConfig{
		DatabaseName: m.client.config.Database,
	})
	defer session.Close()

	_, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(query, params)
		if err != nil {
			return nil, err
		}
		return result.Consume()
	})
	return err
}

// MergeOne 合并单个节点（存在则更新，不存在则创建）
func (m *Model) MergeOne(node interface{}) error {
	// 将单个节点包装成切片调用MergeBatch
	nodes := []interface{}{node}
	return m.MergeBatch(nodes)
}

// MergeOne 批量合并多个节点（存在则更新，不存在则创建）
func (m *Model) MergeBatch(nodes interface{}) error {

	session := m.client.driver.NewSession(neo4j.SessionConfig{
		DatabaseName: m.client.config.Database,
	})
	defer session.Close()

	// 验证输入类型
	nodesValue := reflect.ValueOf(nodes)
	if nodesValue.Kind() != reflect.Slice && nodesValue.Kind() != reflect.Array {
		return fmt.Errorf("%s: expected slice, got %T", ErrInvalidModel, nodes)
	}

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(buildMergeQuery(m, nodesValue))
		if err != nil {
			return nil, fmt.Errorf("merge failed: %w", err)
		}
		return result.Consume()
	})
	return err
}

// buildMergeQuery 构建合并查询（包含节点和关系）
func buildMergeQuery(m *Model, nodesValue reflect.Value) (string, map[string]interface{}) {
	var sb strings.Builder
	params := make(map[string]interface{})

	sb.WriteString("UNWIND $nodes AS node ")
	sb.WriteString("MERGE (n")
	sb.WriteString(":" + m.table)
	sb.WriteString(" { ")
	// 关键修正点：使用node.props访问属性
	sb.WriteString(m.fieldMap[m.primaryKey] + ": node.props." + m.fieldMap[m.primaryKey])
	sb.WriteString("})")

	sb.WriteString("SET n += node.props ")

	// 处理节点参数
	processedNodes := make([]map[string]interface{}, 0, nodesValue.Len())
	for i := 0; i < nodesValue.Len(); i++ {
		node := nodesValue.Index(i).Interface()
		props, err := structToProperties(node)
		if err != nil {
			fmt.Println("structToProperties error: ", err)
			return "", nil
		}
		processedNodes = append(processedNodes, map[string]interface{}{
			"props": props,
		})
	}
	params["nodes"] = processedNodes
	query := sb.String()

	if m.debug {
		fmt.Printf("Executing Merge:\n%s\nWith params: %+v\n", query, params)
	}
	return sb.String(), params
}

// DeleteOne 删除一个节点
func (m *Model) DeleteOne(node interface{}) error {
	// 包装成切片调用DeleteBatch
	nodes := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(node)), 1, 1)
	nodes.Index(0).Set(reflect.ValueOf(node))
	return m.DeleteBatch(nodes.Interface())
}

// DeleteBatch 批量删除节点（包含节点和关系）
func (m *Model) DeleteBatch(nodes interface{}) error {
	nodesValue := reflect.ValueOf(nodes)
	if nodesValue.Kind() != reflect.Slice && nodesValue.Kind() != reflect.Array {
		return fmt.Errorf("%s: expected slice/array, got %T", ErrInvalidModel, nodes)
	}

	session := m.client.driver.NewSession(neo4j.SessionConfig{
		DatabaseName: m.client.config.Database,
	})
	defer session.Close()

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		query, params := buildDeleteQuery(m, nodesValue)
		result, err := tx.Run(query, params)
		if err != nil {
			return nil, fmt.Errorf("delete failed: %w", err)
		}
		return result.Consume()
	})
	return err
}

func buildDeleteQuery(m *Model, nodesValue reflect.Value) (string, map[string]interface{}) {
	var sb strings.Builder
	pks := make([]interface{}, 0, nodesValue.Len())

	// 收集主键值
	for i := 0; i < nodesValue.Len(); i++ {
		node := nodesValue.Index(i)
		if node.Kind() == reflect.Ptr {
			node = node.Elem()
		}
		pkValue := node.FieldByName(m.primaryKey).Interface()
		pks = append(pks, pkValue)
	}

	// 构建Cypher
	sb.WriteString("UNWIND $pks AS pk ")
	sb.WriteString(fmt.Sprintf("MATCH (n:%s) ", m.table))
	sb.WriteString(fmt.Sprintf("WHERE n.%s = pk ", m.fieldMap[m.primaryKey]))
	sb.WriteString(" DETACH DELETE n")
	params := map[string]interface{}{"pks": pks}
	query := sb.String()
	if m.debug {
		fmt.Printf("Executing Merge:\n%s\nWith params: %+v\n", query, params)
	}

	return sb.String(), params
}
