package neo4jorm

import (
	"testing"
)

// type Product struct {
//     SKU  string `neo4j:"name:sku;primary;label:Product"` // 确保主键标签正确
//     Name string `neo4j:"name:product_name"`
//     Price    float64 `neo4j:"name:price"`
//     Stock    int     `neo4j:"name:stock"`
//     Category string  `neo4j:"name:category"`
// }

func TestParseTag(t *testing.T) {
	// 正常情况测试
	tag1 := "key1:value1;key2:value2"
	expected1 := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	result1 := parseTag(tag1)
	if !compareMaps(result1, expected1) {
		t.Errorf("Test case 1 failed: expected %v, got %v", expected1, result1)
	}

	// 只有一个键值对的情况测试
	tag2 := "key:value"
	expected2 := map[string]string{
		"key": "value",
	}
	result2 := parseTag(tag2)
	if !compareMaps(result2, expected2) {
		t.Errorf("Test case 2 failed: expected %v, got %v", expected2, result2)
	}

	// 空标签测试
	tag3 := ""
	expected3 := map[string]string{}
	result3 := parseTag(tag3)
	if !compareMaps(result3, expected3) {
		t.Errorf("Test case 3 failed: expected %v, got %v", expected3, result3)
	}

	// 键或值包含空格的情况测试
	tag4 := "key1 : value1 ; key2 : value2"
	expected4 := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	result4 := parseTag(tag4)
	if !compareMaps(result4, expected4) {
		t.Errorf("Test case 4 failed: expected %v, got %v", expected4, result4)
	}

	// 包含主键和label的情况测试
	tag5 := "name:sku;primary;label:Product"
	expected5 := map[string]string{
		"name":    "sku",
		"primary": "",
		"label":   "Product",
	}
	result5 := parseTag(tag5)
	if !compareMaps(result5, expected5) {
		t.Errorf("Test case 5 failed: expected %v, got %v", expected5, result5)
	}

}

func compareMaps(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}
	for key, value1 := range m1 {
		value2, ok := m2[key]
		if !ok || value1 != value2 {
			return false
		}
	}
	return true
}
