package main

import (
	"fmt"

	"github.com/chengjiahua/neo4jorm"
)

/*
支持的特殊选项列表
选项	说明	示例
primary	主键字段	sku,primary
label	节点标签	label=Product

*/

func main() {
	basicExample()
}

func basicExample() {

	type Product struct {
		SKU      string  `neo4j:"name:sku;primary;label:Product"` // 确保主键标签正确
		Name     string  `neo4j:"name:product_name"`
		Price    float64 `neo4j:"name:price"`
		Stock    *int    `neo4j:"name:stock"`
		Category string  `neo4j:"name:category"`
	}

	// 初始化客户端
	config := &neo4jorm.Config{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "Kylin123.",
		Database: "neo4j",
		Debug:    false,
	}
	orm, err := neo4jorm.NewClient(config)
	if err != nil {
		panic(err)
	}
	defer orm.Close()

	intV := []int{0, 2, 3}
	products := []*Product{
		{SKU: "P1001", Name: "dgsaxvz", Category: "old", Stock: &intV[0], Price: 111.99},
		{SKU: "P1002", Name: "afdf", Price: 1223454}, // 零值示例
	}

	// 执行合并操作
	ProductOrm := orm.Model(&Product{})
	err = ProductOrm.DebugInfo().MergeBatch(products)
	if err != nil {
		panic(err)
	}

	err = ProductOrm.DebugInfo().Update(Product{SKU: "P1001", Category: "new blance", Name: "bbbb", Price: 100.99, Stock: &intV[2]})
	if err != nil {
		panic(err)
	}

	err = ProductOrm.DebugInfo().CreateOne(&Product{SKU: "P1003", Category: "test CreateOne", Name: "aaaa", Price: 0.99, Stock: &intV[1]})
	if err != nil {
		fmt.Println("CreateOne err: ", err)
	}

	err = ProductOrm.DebugInfo().DeleteOne(&Product{SKU: "P1003"})
	if err != nil {
		fmt.Println("DeleteOne err: ", err)
	}

	FindRes := []Product{}
	err = ProductOrm.DebugInfo().Find(&FindRes)
	if err != nil {
		fmt.Println("Find err: ", FindRes)
	}
	fmt.Printf("res:%+v \n", FindRes)

	FindOneRes := Product{}
	err = ProductOrm.Where(Product{SKU: "P1001", Name: "bbbb"}).OrderBy("product_name").Limit(1).DebugInfo().FindOne(&FindOneRes)
	if err != nil {
		fmt.Println("Find err: ", FindOneRes)
	}
	fmt.Printf("res:%+v \n", FindOneRes)
	return
}
