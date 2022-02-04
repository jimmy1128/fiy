package huawei

import (
	"encoding/json"
	"fiy/app/cmdb/models/resource"
	orm "fiy/common/global"
	"fiy/common/log"
	"fiy/pkg/es"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	v2 "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
	region "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/region"
	"gorm.io/gorm/clause"

	"fmt"
)

type huaWeiYun struct {
	SK     string   `json:"sk"`
	AK     string   `json:"ak"`
	Region []string `json:"region"`
}

func NewhuaWeiYun(sk, ak string, region []string) *huaWeiYun{
	return &huaWeiYun{
		SK: sk,
		AK: ak,
		Region: region,
	}
}

func (d *huaWeiYun) EcsList(infoID int,infoName string)(err error){
	var (
		response       *model.ListServersDetailsResponse
		ecsList  []model.ServerDetail
		dataList []resource.Data
		ecsClient *v2.EcsClient
	)
	for _, s := range d.Region {
		auth := basic.NewCredentialsBuilder().
			WithAk(d.AK).
			WithSk(d.SK).
			//WithProjectId("d2100da5212b4007a1ece0a1c9ce31ac").
			Build()
		ecsClient =v2.NewEcsClient(v2.EcsClientBuilder().WithRegion(region.ValueOf(s)).WithCredential(auth).Build())
		limit :=int32(1)
		request := &model.ListServersDetailsRequest{Limit: &limit}
		response, err = ecsClient.ListServersDetails(request)
		if err != nil {
			log.Errorf("查询ECS实例列表失败，%v", err)
			return
		}
		b := int(*response.Count)/100 +1
		if int(*response.Count) > 0{
			for i:=0; i < b;i++{
				request.Offset = Int32Ptr(int32(100*1))
				request.Limit = Int32Ptr(100)
				r,err :=ecsClient.ListServersDetails(request)
				if err != nil {
					log.Errorf("查询ECS实例列表失败，%v", err)
					return err
				}
				ecsList =append(ecsList,*r.Servers...)
			}
		}
	}
	for _, instance:= range ecsList {
		var d []byte
		d, err = json.Marshal(instance)
		if err != nil {
			log.Errorf("序列化服务器数据失败，%v", err)
			return
		}
		tmp := make(map[string]interface{})
		err = json.Unmarshal(d, &tmp)
		if err != nil {
			log.Errorf("反序列化数据失败，", err)
			return
		}
		tmp["instancesID"] = tmp["id"]
		delete(tmp, "id")
		d, err = json.Marshal(tmp)
		if err != nil {
			log.Errorf("序列化服务器数据失败，%v", err)
			return
		}
		dataList = append(dataList, resource.Data{
			Uuid:   fmt.Sprintf("huaweiyun-ecs-%s", instance.Id),
			InfoId: infoID,
			InfoName: infoName,
			Status: 0,
			Data:   d,
		})
	}
	err = orm.Eloquent.Model(&resource.Data{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "uuid"}},
		DoUpdates: clause.AssignmentColumns([]string{"data"}),
	}).Create(&dataList).Error
	//获取数据库的数据
	orm.Eloquent.Model(&resource.Data{}).Where("info_id = ?", infoID).Find(&dataList)
	//添加到es
	err = es.EsClient.Add(dataList)
	if err != nil {
		log.Errorf("索引数据失败，%v", err)
		return
	}
	return

}
func Int32Ptr(v int32) *int32 {
	return &v
}
