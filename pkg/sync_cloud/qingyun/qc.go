package qingyun

import (
	"encoding/json"
	"fiy/app/cmdb/models/resource"
	orm "fiy/common/global"
	"fiy/common/log"
	"fiy/pkg/es"
	"fiy/tools"
	"fmt"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/yunify/qingcloud-sdk-go/config"
	"gorm.io/gorm/clause"

	qc "github.com/yunify/qingcloud-sdk-go/service"
)
type qingCloud struct {
	SK     string   `json:"sk"`
	AK     string   `json:"ak"`
	Region []string `json:"region"`
}

func NewQingYun(sk,ak string,region []string)*qingCloud{
	return &qingCloud{
		SK: sk,
		AK: ak,
		Region: region,
	}
}

func (f *qingCloud) QcList(infoID int,infoName string)(err error){
	var (
		qcList []*qc.Instance
		dataList  []resource.Data
		response  *qc.DescribeInstancesOutput
		qcService *qc.QingCloudService
		gcInstance *qc.InstanceService
	)

	configuration,err := config.New(tools.Strip(f.SK), tools.Strip(f.AK))
	if err != nil {
		log.Errorf("创建客户端连接失败，%v", err)
	}
	configuration.Host = "api.greyconsole.com"
	configuration.Protocol = "https"
	configuration.Port = 443
	qcService, err = qc.Init(configuration)
	if err != nil {
		log.Errorf("创建客户端连接失败，%v", err)
	}
	for _, s := range f.Region {
		gcInstance, err = qcService.Instance(tools.Strip(s))
		if err != nil {
			log.Fatal(err)
		}
		args := []string{"pending","running","stopped","suspended","rescuing"}
		response, _ =gcInstance.DescribeInstances(&qc.DescribeInstancesInput{

			Limit: common.IntPtr(100),
			Offset: common.IntPtr(0),
			Owner: common.StringPtr("usr-xGPBLqoH"),
			Verbose: common.IntPtr(1),
			Status:common.StringPtrs(args),
		})
		if *response.TotalCount > 0 {
			for i:=0 ; i<*response.TotalCount/100 +1 ;i++{
				response, _ =gcInstance.DescribeInstances(&qc.DescribeInstancesInput{
					Limit: common.IntPtr(100),
					Offset: common.IntPtr(100 * i),
					Owner: common.StringPtr("usr-xGPBLqoH"),
					Verbose: common.IntPtr(1),
					Status:common.StringPtrs(args),
				})
				qcList = append(qcList, response.InstanceSet...)
			}
		}
		for _, instance := range qcList {
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
			if *instance.EIP.EIPID != ""{
				tmp["bandwidth"] = *instance.EIP.Bandwidth
				tmp["eip_addr"]= *instance.EIP.EIPAddr
				tmp["eip_id"] = *instance.EIP.EIPID
			}
			delete(tmp,"security_group")
			delete(tmp,"security_groups")
			delete(tmp,"image")
			delete(tmp,"extra")
			delete(tmp,"eip")
			d, err = json.Marshal(tmp)
			if err != nil {
				log.Errorf("序列化服务器数据失败，%v", err)
				return
			}
			dataList = append(dataList, resource.Data{
				Uuid:   fmt.Sprintf("qingyun-qc-%s", *instance.InstanceID),
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
		orm.Eloquent.Model(&resource.Data{}).Where("info_id = ?", infoID).Find(&dataList)
		err = es.EsClient.Add(dataList)
		if err != nil {
			log.Errorf("索引数据失败，%v", err)
			return
		}
	}
	return
}

func (f *qingCloud) QcIpList(infoID int,infoName string)(err error){
	var (
		qcIpList []*qc.EIP
		dataList  []resource.Data
		response  *qc.DescribeEIPsOutput
		qcService *qc.QingCloudService
		gcEip *qc.EIPService
	)

	configuration,err := config.New(tools.Strip(f.SK), tools.Strip(f.AK))
	if err != nil {
		log.Errorf("创建客户端连接失败，%v", err)
	}
	configuration.Host = "api.greyconsole.com"
	configuration.Protocol = "https"
	configuration.Port = 443
	qcService, err = qc.Init(configuration)
	if err != nil {
		log.Errorf("创建客户端连接失败，%v", err)
	}
	for _, s := range f.Region {
		gcEip, err = qcService.EIP(tools.Strip(s))
		if err != nil {
			log.Fatal(err)
		}
		inuse := []string{"pending","available","associated","suspended"} // "pending","available","associated","suspended"
		response, _ = gcEip.DescribeEIPs(&qc.DescribeEIPsInput{
			Owner: common.StringPtr("usr-xGPBLqoH"),
			Status:common.StringPtrs(inuse),
		})

		if *response.TotalCount > 0 {
			for i:=0 ; i<*response.TotalCount/100 +1 ;i++{
				response, _ =gcEip.DescribeEIPs(&qc.DescribeEIPsInput{
					Limit: common.IntPtr(100),
					Offset: common.IntPtr(100 * i),
					Owner: common.StringPtr("usr-xGPBLqoH"),
					Verbose: common.IntPtr(1),
					Status:common.StringPtrs(inuse),
				})
				qcIpList = append(qcIpList, response.EIPSet...)
			}
		}

		for _, instance := range qcIpList {
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

			tmp["eip_group_name"]= *instance.EIPGroup.EIPGroupName
			//tmp["nic_id"]= *instance.Resource
			tmp["resource_id"]= *instance.Resource.ResourceID

			delete(tmp,"billing_mode")
			delete(tmp,"eip_group")
			delete(tmp,"resource")
			d, err = json.Marshal(tmp)
			if err != nil {
				log.Errorf("序列化服务器数据失败，%v", err)
				return
			}
			dataList = append(dataList, resource.Data{
				Uuid:   fmt.Sprintf("qingyun-qc-%s-%s", *instance.Resource.ResourceID,*instance.EIPID),
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
		orm.Eloquent.Model(&resource.Data{}).Where("info_id = ?", infoID).Find(&dataList)
		err = es.EsClient.Add(dataList)
		if err != nil {
			log.Errorf("索引数据失败，%v", err)
			return
		}
	}
return
}
