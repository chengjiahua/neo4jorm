# Neo4j ORM

## 项目简介

笔者学习 Neo4j 的过程中，发现官方的 Golang 驱动原生操作不好用，因此笔者决定自己写一个简单的 ORM 工具，希望能够帮助到更多的人。

Neo4j ORM 是一个用于与 Neo4j 数据库交互的对象关系映射（ORM）工具。它简化了 Neo4j 数据库的操作，使开发者能够使用面向对象的方式进行数据库操作。

## 安装

请确保您的系统已经安装了 Golang。然后运行以下命令来安装依赖项：

因为使用了 merge 操作，需要使用 neo4j4.4+及以上版本

如果你的 neo4j 是 v5 版本，需要使用以下命令安装 neo4j 驱动

```bash
go get github.com/neo4j/neo4j-go-driver/v5/neo4j
```

如果你的 neo4j 是 v4 版本，需要使用以下命令安装 neo4j 驱动

```bash
go get github.com/neo4j/neo4j-go-driver/v4/neo4j
```

```bash
go get github.com/chengjiahua/neo4jorm
```

## 使用方法

以下是一个简单的使用示例，更多示例请参考`examples`目录：

```go
/*
支持的特殊选项列表
选项	说明	示例
primary	主键字段	sku,primary
label	节点标签	label=Product
name    tagkey,对应neo4j的标签名
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


```

## 贡献

欢迎贡献代码！请提交 Pull Request 或报告问题。

## 许可证

本项目采用 MIT 许可证。

## 工作计划

```
- [2025/2/28] 支持写入节点操作
- [2025/3/1] 支持批量写入节点，灵活更新节点操作
- [2025/3/2] 支持读取单个节点，多个节点操作
- [2025/3/2] 支持merge(新增,更新)，删除节点关系操作（因为cypher不支持无方向的关系，暂时用双向关系忽略关系方向）
- [x] 支持读取节点以及关系操作
- [x] 支持读取链路信息，通用接口根据path查询。
- [x] 支持批量读写灵活写入节点及关系操作，做成类似gorm调用方式
- [x] 支持事务
- [x] 支持负载均衡
```
