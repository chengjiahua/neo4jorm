package neo4jorm

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type Query struct {
	model      *Model
	cypher     strings.Builder
	params     map[string]interface{}
	limit      int
	skip       int
	orderBy    string
	returnExpr string
}

func (q *Query) Where(condition string, params map[string]interface{}) *Query {
	if q.params == nil {
		q.params = make(map[string]interface{})
	}

	// 参数合并
	for k, v := range params {
		q.params[k] = v
	}

	if strings.Contains(q.cypher.String(), "WHERE") {
		q.cypher.WriteString(" AND " + condition)
	} else {
		q.cypher.WriteString(" WHERE " + condition)
	}
	return q
}

func (q *Query) Limit(limit int) *Query {
	q.limit = limit
	return q
}

func (q *Query) Skip(skip int) *Query {
	q.skip = skip
	return q
}

func (q *Query) OrderBy(field string) *Query {
	q.orderBy = field
	return q
}

func (q *Query) Return(expr string) *Query {
	q.returnExpr = expr
	return q
}

func (q *Query) Exec() ([]interface{}, error) {
	session := q.model.client.driver.NewSession(neo4j.SessionConfig{
		DatabaseName: q.model.client.config.Database,
	})
	defer session.Close()

	// 构建最终Cypher
	var finalQuery strings.Builder
	finalQuery.WriteString("MATCH (n")
	for _, label := range q.model.labels {
		finalQuery.WriteString(":" + label)
	}
	finalQuery.WriteString(")\n")
	finalQuery.WriteString(q.cypher.String())

	if q.orderBy != "" {
		finalQuery.WriteString(fmt.Sprintf(" ORDER BY %s", q.orderBy))
	}
	if q.skip > 0 {
		finalQuery.WriteString(fmt.Sprintf(" SKIP %d", q.skip))
	}
	if q.limit > 0 {
		finalQuery.WriteString(fmt.Sprintf(" LIMIT %d", q.limit))
	}
	if q.returnExpr != "" {
		finalQuery.WriteString(fmt.Sprintf(" RETURN %s", q.returnExpr))
	} else {
		finalQuery.WriteString(" RETURN n")
	}

	result, err := session.Run(finalQuery.String(), q.params)
	if err != nil {
		return nil, err
	}

	var nodes []interface{}
	for result.Next() {
		record := result.Record()
		if value, ok := record.Get("n"); ok {
			if node, ok := value.(neo4j.Node); ok {
				newNode := reflect.New(q.model.modelType).Interface()
				if err := decodeNode(node, newNode); err != nil {
					return nil, err
				}
				nodes = append(nodes, newNode)
			}
		}
	}
	return nodes, nil
}