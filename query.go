package neo4jorm

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

// Where 添加查询条件
func (m *Model) Where(condition interface{}, args ...interface{}) *Model {
	m.params = make(map[string]interface{})
	// 类型反射处理
	condVal := reflect.ValueOf(condition)
	if condVal.Kind() == reflect.Ptr {
		condVal = condVal.Elem()
	}

	// 类型匹配检查
	if condVal.IsValid() && condVal.Type() == m.modelType {
		// 处理同类型结构体条件
		var conditions []string
		params := make(map[string]interface{})

		for i := 0; i < condVal.NumField(); i++ {
			field := m.modelType.Field(i)
			fieldVal := condVal.Field(i)

			// 跳过零值字段
			if isZeroValue(fieldVal) {
				continue
			}

			// 获取映射后的属性名
			propName := m.fieldMap[field.Name]

			// 构造条件表达式
			paramKey := fmt.Sprintf("%s_%d", propName, len(m.params))
			conditions = append(conditions, fmt.Sprintf("n.%s = $%s", propName, paramKey))
			params[paramKey] = fieldVal.Interface()
		}

		if len(conditions) > 0 {
			m.conditions = append(m.conditions, strings.Join(conditions, " AND "))
			for k, v := range params {
				m.params[k] = v
			}
		}
	} else {
		// 处理字符串条件
		switch c := condition.(type) {
		case string:
			m.conditions = append(m.conditions, c)
			if len(args) > 0 {
				if params, ok := args[0].(map[string]interface{}); ok {
					for k, v := range params {
						m.params[k] = v
					}
				}
			}
		}
	}

	return m
}

// OrderBy 添加排序条件
func (m *Model) OrderBy(fields ...string) *Model {
	m.orderBy = append(m.orderBy, fmt.Sprintf(" '%s' ", strings.Join(fields, "','")))
	return m
}

// Limit 设置结果数量限制
func (m *Model) Limit(limit int) *Model {
	m.limit = limit
	return m
}

// buildQuery 构建Cypher查询语句
func (m *Model) buildQuery() string {
	var query strings.Builder
	query.WriteString(fmt.Sprintf("MATCH (n:%s)", m.table))

	// 处理WHERE条件
	if len(m.conditions) > 0 {
		query.WriteString(" WHERE " + strings.Join(m.conditions, " AND "))
	}
	query.WriteString(" RETURN n ")
	// 处理ORDER BY
	if len(m.orderBy) > 0 {
		query.WriteString(" ORDER BY " + strings.Join(m.orderBy, ", "))
	}

	// 处理LIMIT
	if m.limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d", m.limit))
	}

	return query.String()
}

// FindOne 查询单个结果
func (m *Model) FindOne(result interface{}) error {
	return m.executeQuery(m.Limit(1).buildQuery(), result, true)
}

// Find 查询多个结果
func (m *Model) Find(results interface{}) error {
	return m.executeQuery(m.buildQuery(), results, false)
}

// executeQuery 执行查询并处理结果映射
func (m *Model) executeQuery(query string, out interface{}, single bool) error {
	session := m.client.driver.NewSession(neo4j.SessionConfig{
		DatabaseName: m.client.config.Database,
	})
	defer session.Close()

	if m.debug {
		fmt.Printf("Executing Query:\n%s\nWith params: %+v\n", query, m.params)
	}

	result, err := session.Run(query, m.params)
	if err != nil {
		return err
	}

	// 校验输出类型
	outVal := reflect.ValueOf(out)
	if outVal.Kind() != reflect.Ptr {
		return errors.New("output must be a pointer")
	}

	// 处理结果集
	sliceVal := outVal.Elem()
	if !single && sliceVal.Kind() != reflect.Slice {
		return errors.New("results must be a pointer to a slice")
	}

	for result.Next() {
		record := result.Record()
		node, ok := record.GetByIndex(0).(neo4j.Node)
		if !ok {
			return errors.New("query did not return a node")
		}

		// 创建新实例并映射属性
		elem := reflect.New(m.modelType).Interface()
		if err := m.mapToStruct(node.Props, elem); err != nil {
			return err
		}

		if single {
			outVal.Elem().Set(reflect.ValueOf(elem).Elem())
			return nil // 找到即返回
		} else {
			sliceVal.Set(reflect.Append(sliceVal, reflect.ValueOf(elem).Elem()))
		}
	}

	if err := result.Err(); err != nil {
		return err
	}

	if single {
		return errors.New("no records found")
	}
	// 查询玩后将参数清零，避免影响下次查询
	m.cleanQuery()
	return nil
}

func (m *Model) cleanQuery() {
	m.conditions = nil
	m.params = nil
	m.orderBy = nil
	m.limit = 0
}

// mapToStruct 将节点属性映射到结构体
func (m *Model) mapToStruct(properties map[string]interface{}, result interface{}) error {
	resultVal := reflect.ValueOf(result).Elem()

	for i := 0; i < resultVal.NumField(); i++ {
		field := resultVal.Type().Field(i)
		fieldVal := resultVal.Field(i)

		// 获取映射属性名
		propName := m.fieldMap[field.Name]
		if propName == "" {
			propName = field.Name
		}

		// 获取属性值
		value, exists := properties[propName]
		if !exists {
			continue // 属性不存在时跳过
		}

		// 处理指针类型的特殊逻辑
		if fieldVal.Kind() == reflect.Ptr {
			if value == nil {
				fieldVal.Set(reflect.Zero(fieldVal.Type())) // 设置为nil指针
				continue
			}

			// 创建新的指针并赋值
			elemType := fieldVal.Type().Elem() // 获取指针指向的类型

			// 根据指针指向的类型进行转换
			switch elemType.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if v, ok := convertToInt(reflect.ValueOf(value)); ok {
					ptr := reflect.New(elemType)
					ptr.Elem().SetInt(v)
					fieldVal.Set(ptr)
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				if v, ok := convertToUint(reflect.ValueOf(value)); ok {
					ptr := reflect.New(elemType)
					ptr.Elem().SetUint(v)
					fieldVal.Set(ptr)
				}
			case reflect.Float32, reflect.Float64:
				if v, ok := convertToFloat(reflect.ValueOf(value)); ok {
					ptr := reflect.New(elemType)
					ptr.Elem().SetFloat(v)
					fieldVal.Set(ptr)
				}
			case reflect.String:
				if s, ok := value.(string); ok {
					ptr := reflect.New(elemType)
					ptr.Elem().SetString(s)
					fieldVal.Set(ptr)
				}
			default:
				return fmt.Errorf("不支持的指针类型: %s", elemType.Kind())
			}
			continue
		}

		// 处理空值
		if value == nil {
			if fieldVal.Kind() == reflect.Ptr {
				fieldVal.Set(reflect.Zero(fieldVal.Type())) // 设置指针为nil
			} else {
				// 根据字段类型设置零值
				zeroVal := reflect.Zero(fieldVal.Type())
				if fieldVal.CanSet() {
					fieldVal.Set(zeroVal)
				}
			}
			continue
		}

		// 类型转换处理
		val := reflect.ValueOf(value)
		if val.Type().AssignableTo(fieldVal.Type()) {
			fieldVal.Set(val)
		} else if val.Type().ConvertibleTo(fieldVal.Type()) {
			convertedVal := val.Convert(fieldVal.Type())
			fieldVal.Set(convertedVal)
		} else {
			// 处理常见类型不匹配情况
			switch fieldVal.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if v, ok := convertToInt(val); ok {
					fieldVal.SetInt(v)
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				if v, ok := convertToUint(val); ok {
					fieldVal.SetUint(v)
				}
			case reflect.Float32, reflect.Float64:
				if v, ok := convertToFloat(val); ok {
					fieldVal.SetFloat(v)
				}
			case reflect.String:
				fieldVal.SetString(fmt.Sprintf("%v", value))
			default:
				return fmt.Errorf("字段 %s 类型不匹配 (数据库类型: %T, 结构体类型: %s)",
					field.Name, value, fieldVal.Type().String())
			}
		}
	}
	return nil
}

// FindByPrimaryKey 根据主键查询
func (m *Model) FindByPrimaryKey(value interface{}, result interface{}) error {
	if m.primaryKey == "" {
		return errors.New("primary key not defined")
	}
	propName := m.fieldMap[m.primaryKey]
	return m.Where(fmt.Sprintf("n.%s = $pk", propName), map[string]interface{}{"pk": value}).FindOne(result)
}
