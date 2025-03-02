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

	err = basicExample(orm)
	if err != nil {
		fmt.Println("basicExample err: ", err)
	}
	// err = relationshipExample(orm)
	// if err != nil {
	// 	fmt.Println("relationshipExample err: ", err)
	// }

}

func basicExample(orm *neo4jorm.Client) error {

	type Product struct {
		ID       int64   `neo4j:"name=id,table=Product"`
		SKU      string  `neo4j:"name=sku,primary"` // 确保主键标签正确
		Name     string  `neo4j:"name=product_name"`
		Price    float64 `neo4j:"name=price"`
		Stock    *int    `neo4j:"name=stock"`
		Category string  `neo4j:"name=category"`
	}

	intV := []int{0, 2, 3}
	products := []*Product{
		{SKU: "P1001", Name: "dgsaxvz", Category: "old", Stock: &intV[0], Price: 111.99},
		{SKU: "P1002", Name: "afdf", Price: 1223454}, // 零值示例
	}
	// 执行合并操作
	ProductOrm := orm.Model(&Product{})
	err := ProductOrm.DebugInfo().DeleteBatch([]*Product{{SKU: "P1003"}, {SKU: "P1002"}, {SKU: "P1001"}})
	if err != nil {
		fmt.Println("DeleteBatch err: ")
	}

	err = ProductOrm.DebugInfo().CreateOne(&Product{SKU: "P1003", Category: "test CreateOne", Name: "aaaa", Price: 0.99, Stock: &intV[1]})
	if err != nil {
		fmt.Println("CreateOne err: ", err)
	}

	err = ProductOrm.DebugInfo().MergeBatch(products)
	if err != nil {
		panic(err)
	}

	err = ProductOrm.DebugInfo().Update(Product{SKU: "P1001", Category: "new blance", Name: "bbbb", Price: 100.99, Stock: &intV[2]})
	if err != nil {
		panic(err)
	}

	// 每个Relation的start表都是table1，end表都是table2
	err = ProductOrm.DebugInfo().CreateRelations([]neo4jorm.Relation{
		{Start: &Product{SKU: "P1001"}, End: &Product{SKU: "P1002"}},
	}, "RELATION")
	if err != nil {
		fmt.Println("CreateRelation err: ", err)
	}

	err = ProductOrm.DebugInfo().DeleteRelations([]neo4jorm.Relation{
		{Start: &Product{SKU: "P1001"}, End: &Product{SKU: "P1002"}},
	}, "RELATION")
	if err != nil {
		fmt.Println("DeleteRelations err: ", err)
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

	return err
}

func relationshipExample(orm *neo4jorm.Client) error {
	type Project struct {
		ID    string `neo4j:"name:id,primary,label=Project"`
		Title string `neo4j:"name:title"`
		// 关系属性
		Role string `neo4j:"name:role"` // 这个字段会作为关系属性
	}

	type User struct {
		ID   string `neo4j:"id,primary,label=User"`
		Name string `neo4j:"name"`
		// 1:1 关系
		Manager *User `neo4j:"rel=REPORTS_TO,direction=outgoing,merge=true"`
		// 1:N 关系
		Friends []*User `neo4j:"rel=FRIENDS,direction=both,merge=true"`
		// 带属性的关系
		Projects []*Project `neo4j:"rel=OWNS,direction=outgoing,merge=true"`
	}

	manager := &User{ID: "M001", Name: "Alice"}
	project := &Project{ID: "P001", Title: "Neo4j ORM", Role: "Owner"}

	user := &User{
		ID:      "U001",
		Name:    "Bob",
		Manager: manager,
		Friends: []*User{
			{ID: "U002", Name: "Charlie"},
			{ID: "U003", Name: "David"},
		},
		Projects: []*Project{project},
	}

	// 自动合并节点和关系
	err := orm.Model(&User{}).MergeBatch([]*User{user})
	if err != nil {
		panic(err)
	}

	return nil
}
