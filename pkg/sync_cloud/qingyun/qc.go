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
	"strings"
	"time"

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
				Uuid:   fmt.Sprintf("qingyun-qc-(%s)", *instance.InstanceID),
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
		dataList1  []resource.Data
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
			delete(tmp,"status")
			tmp["eip_group_name"]= *instance.EIPGroup.EIPGroupName
			tmp["resource_id"]= *instance.Resource.ResourceID
			if *instance.Status == "associated"{
				tmp["status"] = "使用中"
			}else if *instance.Status == "available"{
				tmp["status"] = "空"
			}
			delete(tmp,"billing_mode")
			delete(tmp,"eip_group")
			delete(tmp,"resource")
			d, err = json.Marshal(tmp)
			if err != nil {
				log.Errorf("序列化服务器数据失败，%v", err)
				return
			}
			dataList1 = append(dataList1, resource.Data{
				Uuid:   fmt.Sprintf("qingyun-qc-(%s)",*instance.EIPID),
				Instance: *instance.Resource.ResourceID,
				InfoId: infoID,
				InfoName: infoName,
				Status: 0,
				Data:   d,
			})
		}
		err = orm.Eloquent.Model(&resource.Data{}).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "uuid"}},
			DoUpdates: clause.AssignmentColumns([]string{"data"}),
		}).Create(&dataList1).Error
		orm.Eloquent.Model(&resource.Data{}).Where("info_id = ?", infoID).Find(&dataList1)
		err = es.EsClient.Add(dataList1)
		if err != nil {
			log.Errorf("索引数据失败，%v", err)
			return
		}
	}
return
}

func QcAttachIp (sourceId int , targetId int , targetInfoId int)(err error) {
	var (

		qcService *qc.QingCloudService
		gcNic *qc.NicService
		source resource.Data
		target resource.Data
		account resource.CloudAccount
		discovery_account resource.CloudDiscovery
	)
	err = orm.Eloquent.Model(&resource.CloudDiscovery{}).
		Where("resource_model= ?",targetInfoId).
		Find(&discovery_account).Error
	err = orm.Eloquent.Model(&resource.CloudDiscovery{}).
		Where("resource_model= ?",discovery_account.CloudAccount).
		Find(&account).Error

	err = orm.Eloquent.Model(&resource.Data{}).
		Where("id = ? ", sourceId).
		Find(&source).Error
	err = orm.Eloquent.Model(&resource.Data{}).
		Where("id = ? ", targetId).
		Find(&target).Error

	configuration,err := config.New(tools.Strip(account.Secret), tools.Strip(account.Key))
	if err != nil {
		log.Errorf("创建客户端连接失败，%v", err)
	}
	configuration.Host = "api.greyconsole.com"
	configuration.Protocol = "https"
	configuration.Port = 443
	//官方api
	qcService, err = qc.Init(configuration)
	// 自定义api
	qcServiceS, err := Init(configuration)
	regionList := make([]string, 0)
	err = json.Unmarshal(discovery_account.Region, &regionList)
	for _, s := range regionList {

		gcNic, err = qcService.Nic(tools.Strip(s))
		gceips, err := qcServiceS.EIPS(tools.Strip(s))
		if err != nil {
			log.Errorf("创建客户端连接失败，%v", err)
		}
		iOutputeip, _ := gcNic.DescribeNics(&qc.DescribeNicsInput{
			Owner:  common.StringPtr("usr-xGPBLqoH"),
			Status: common.StringPtr("available"),
			Limit:  common.IntPtr(1),
		})
		for _, nic := range iOutputeip.NICSet {
			args := []string{*nic.NICID}
			Str := common.StringPtr(format(source.Uuid))
			iOutputenics, _ := gcNic.AttachNics(&qc.AttachNicsInput{Instance: Str, Nics: common.StringPtrs(args)})
			retCode := common.IntPtr(0)
			time.Sleep(5 * time.Second)
			if *iOutputenics.RetCode == *retCode {

				iOutputeipss, _ := gceips.AssociateEIPss(&AssociateEIPInput{
					EIP:      common.StringPtr(format(target.Uuid)),
					Instance: Str,
					Nic:      nic.NICID, //common.StringPtr("52:54:ca:04:0d:30")
				})
				fmt.Println(*iOutputeipss.RetCode)
			}

		}
	}
return

}
func (f *qingCloud)GcAutoRelate (infoID int,infoName string)(err error){
	var (
		dataList  []resource.Data
		relatedList []resource.DataRelated
		dataSource  []resource.Data

	)
	orm.Eloquent.Model(&resource.Data{}).Where("info_id = ?", infoID).Find(&dataList)

	for _, data := range dataList {
		orm.Eloquent.Model(&resource.Data{}).Where("instance = ?", format(data.Uuid)).Find(&dataSource)
		for _, r := range dataSource {

			relatedList = make([]resource.DataRelated, 0)
			relatedList = append(relatedList, resource.DataRelated{
				Source:      data.Id,
				Target:       r.Id,
				SourceInfoId: infoID,
				TargetInfoId: r.InfoId,
			})
			err = orm.Eloquent.Create(&relatedList).Error
			if err != nil {
				log.Errorf("创建数据关联失败，%v", err)
				return
			}
		}

	}
	return
}

func format (string2 string) string {
	s := strings.Split(string2,"(")
	i := strings.Split(s[1], ")")
	return i[0]

}
