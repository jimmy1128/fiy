package es

import (
	"context"
	"fiy/app/cmdb/models/resource"
	fiyLog "fiy/common/log"
	"fmt"
	"github.com/olivere/elastic/v7"
	"github.com/spf13/viper"
	"log"
	"os"
	"time"
)

/*
  @Author : lanyulei
*/

type EsClientType struct {
	EsClient *elastic.Client

}

var EsClient EsClientType //连接类型

func Init() {
	//es 配置
	var (
		err            error
		host           = viper.GetString("settings.es.host")
		esClientParams []elastic.ClientOptionFunc
	)
	esClientParams = []elastic.ClientOptionFunc{

		elastic.SetURL(host),
		elastic.SetSniff(false),
		elastic.SetHealthcheckInterval(10 * time.Second),
		elastic.SetGzip(true),
		elastic.SetErrorLog(log.New(os.Stderr, "ELASTIC ", log.LstdFlags)),
		elastic.SetInfoLog(log.New(os.Stdout, "", log.LstdFlags)),

	}

	if viper.GetString("settings.es.user") != "" && viper.GetString("settings.es.password") != "" {
		esClientParams = append(esClientParams, elastic.SetBasicAuth(viper.GetString("settings.es.user"), viper.GetString("settings.es.password")))
	}

	EsClient.EsClient, err = elastic.NewClient(esClientParams...)

	if err != nil {
		fiyLog.Fatal(err)
	}

	info, code, err := EsClient.EsClient.Ping(host).Do(context.Background())
	if err != nil {
		fiyLog.Fatal(err)
	}

	fiyLog.Infof("Elasticsearch returned with code %d and version %s\n", code, info.Version.Number)

	esVersion, err := EsClient.EsClient.ElasticsearchVersion(host)
	if err != nil {
		fiyLog.Fatal(err)
	}


	fiyLog.Infof("Elasticsearch version %s\n", esVersion)
	fiyLog.Info("connect es success，", EsClient.EsClient)
}

//搜索
func (e EsClientType) Query(value interface{}, page int, limit int) (searchResult *elastic.SearchResult, err error) {
	queryString := elastic.NewQueryStringQuery(fmt.Sprintf("*%v*", value))

	searchResult, err = e.EsClient.Search().
		Index(viper.GetString("settings.es.index")). // 设置索引名
		Query(queryString).                          // 设置查询条件
		From(page).                                  // 设置分页参数 - 起始偏移量，从第0行记录开始
		Size(limit).                                 // 设置分页参数 - 每页大小
		Pretty(true).                                // 查询结果返回可读性较好的JSON格式
		Do(context.Background())                     // 执行请求

	if err != nil {
		fiyLog.Errorf("查询资源数据失败，", err)
		return
	}

	fiyLog.Infof("查询消耗时间 %d ms, 结果总数: %d\n", searchResult.TookInMillis, searchResult.TotalHits())
	return
}
//索引
func (e EsClientType) Add (dataList  []resource.Data)(err error){
	ctx := context.Background()

	exists, err := e.EsClient.IndexExists("cmdb_resource_data").Do(ctx)
	if err != nil {
		// Handle error
		panic(err)
	}
	if !exists {
		// cmdb_resource_data索引不存在，则创建一个
		_, err := e.EsClient.CreateIndex("cmdb_resource_data").Do(ctx)
		if err != nil {
			// Handle error
			panic(err)
		}
	}
	var d resource.Data

	for _, data := range dataList {
		d.Id = data.Id
		d.InfoId = data.InfoId
		d.InfoName = data.InfoName
		d.Data = data.Data

		e.EsClient.Index().Index("cmdb_resource_data").Id(string(d.Id)).BodyJson(d).Do(ctx)

	}

return err
}
