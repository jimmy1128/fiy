package sync_cloud

import (
	"encoding/json"
	"fiy/pkg/sync_cloud/azure"
	"fiy/pkg/sync_cloud/baidu"
	"fiy/pkg/sync_cloud/huawei"
	"fiy/pkg/sync_cloud/qingyun"
	"fiy/pkg/sync_cloud/tencent"
	"fmt"
	"time"

	"fiy/common/log"

	"fiy/pkg/sync_cloud/aliyun"

	"fiy/app/cmdb/models/resource"
	orm "fiy/common/global"

	"github.com/spf13/viper"
)

/*
  @Author : lanyulei
*/

type syncStatus struct {
	ID     int  `json:"id"`
	Status bool `json:"status"`
}

type cloudInfo struct {
	resource.CloudDiscovery
	AccountName   string `json:"account_name"`
	AccountType   string `json:"account_type"`
	AccountStatus bool   `json:"account_status"`
	AccountSecret string `json:"account_secret"`
	AccountKey    string `json:"account_key"`
	AccountTenant    string `json:"account_tenant"`
	AccountSubscript   string `json:"account_subscript"`
}

// 执行同步任务
func syncCloud() (err error) {

	var (
		ch                 chan syncStatus
		cloudDiscoveryList []*cloudInfo
	)
	// 查询所有的任务列表
	err = orm.Eloquent.Model(&resource.CloudDiscovery{}).
		Joins("left join cmdb_resource_cloud_account as crca on crca.id = cmdb_resource_cloud_discovery.cloud_account").
		Select("cmdb_resource_cloud_discovery.*, crca.name as account_name, crca.type as account_type, crca.status as account_status, crca.secret as account_secret, crca.key as account_key,crca.tenant as account_tenant, crca.subscript as account_subscript").
		Where("crca.status = ? and cmdb_resource_cloud_discovery.status = ?", true, true).
		Find(&cloudDiscoveryList).Error
	if err != nil {
		log.Errorf("查询所有同步任务列表失败", err)
		return
	}

	ch = make(chan syncStatus, len(cloudDiscoveryList))
	// 接受云资产同步任务执行结果，并处理
	go func(c <-chan syncStatus) {
		defer close(ch)
		for i := 0; i < len(cloudDiscoveryList); i++ {
			r := <-ch
			err = orm.Eloquent.Model(&resource.CloudDiscovery{}).
				Where("id = ?", r.ID).
				Updates(map[string]interface{}{
					"last_sync_status": r.Status,
					"last_sync_time":   time.Now().Format("2006-01-02 15:04:05"),
				}).Error
			if err != nil {
				log.Errorf("更新同步任务执行状态失败", err)
				return
			}
		}
	}(ch)

	// 开启多个goroutine执行云资源任务同步
	for _, task := range cloudDiscoveryList {
		go func(t *cloudInfo, c chan<- syncStatus) {
			defer func(t1 *cloudInfo) {
				if err := recover(); err != nil {
					c <- syncStatus{
						ID:     t1.Id,
						Status: false,
					}
				}
			}(t)

			var err error

			if t.AccountType == "aliyun" {
				regionList := make([]string, 0)
				err = json.Unmarshal(t.Region, &regionList)

				aLiYunClient := aliyun.NewALiYun(t.AccountSecret, t.AccountKey, regionList)
				if t.ResourceType == 1 { // 查询云主机资产
					err = aLiYunClient.EcsList(t.ResourceModel,t.Name)
				}

				if err != nil {
					errValue := fmt.Sprintf("同步阿里云资源失败，%v", err)
					log.Error(errValue)
					panic(errValue)
				} else {
					c <- syncStatus{
						ID:     t.Id,
						Status: true,
					}
				}

			} else if t.AccountType == "baidu" {
				regionList := make([]string, 0)
				err = json.Unmarshal(t.Region, &regionList)

				baiDuYunClient := baidu.NewBaiDuYun(t.AccountSecret, t.AccountKey, regionList)
				if t.ResourceType == 1 { // 查询云主机资产
					err = baiDuYunClient.BccList(t.ResourceModel,t.Name)
				}

				if err != nil {
					errValue := fmt.Sprintf("同步百度云资源失败，%v", err)
					log.Error(errValue)
					panic(errValue)
				} else {
					c <- syncStatus{
						ID:     t.Id,
						Status: true,
					}
				}
			}else if t.AccountType == "tencent" {
				regionList := make([]string, 0)
				err = json.Unmarshal(t.Region,&regionList)
				tenCentYunClient := tencent.NewTencentYun(t.AccountSecret,t.AccountKey,regionList)
				if t.ResourceType == 1 {
					err = tenCentYunClient.TccList(t.ResourceModel,t.Name)

				}
				if err != nil {
					errValue := fmt.Sprintf("同步腾讯云资源失败，%v", err)

					log.Error(errValue)
					panic(errValue)
				}else {
					c <- syncStatus{
						ID: t.Id,
						Status: true,
					}
				}
			}else if t.AccountType == "huawei" {
				regionList := make([]string, 0)
				err = json.Unmarshal(t.Region,&regionList)
				huaWeiYunClient := huawei.NewhuaWeiYun(t.AccountSecret,t.AccountKey,regionList)
				if t.ResourceType == 1 {
					err = huaWeiYunClient.EcsList(t.ResourceModel,t.Name)
				}
				if err != nil {
					errValue := fmt.Sprintf("同步华为云资源失败，%v", err)
					log.Error(errValue)
					panic(errValue)
				}else {
					c <- syncStatus{
						ID: t.Id,
						Status: true,
					}
				}
			}else if t.AccountType == "azure" {
				regionList := make([]string, 0)
				err = json.Unmarshal(t.Region,&regionList)
				azureCloudClient := azure.NewAzure(t.AccountSecret,t.AccountTenant,t.AccountSubscript,t.AccountKey,regionList)
				if t.ResourceType == 1 {
					err = azureCloudClient.ArmList(t.ResourceModel,t.Name)
					err = azureCloudClient.ArmAutoRelate(t.ResourceModel,t.Name)
				}else if t.ResourceType == 2{
					err = azureCloudClient.ArmNetworkList(t.ResourceModel,t.Name)
				}
				if err != nil {
					errValue := fmt.Sprintf("同步微软云资源失败，%v", err)
					log.Error(errValue)
					panic(errValue)
				}else {
					c <- syncStatus{
						ID: t.Id,
						Status: true,
					}
				}
			}else if t.AccountType == "qingYun"{
				regionList := make([]string,0)
				err = json.Unmarshal(t.Region,&regionList)
				qingCloudClient := qingyun.NewQingYun(t.AccountSecret,t.AccountKey,regionList)
				if t.ResourceType == 1 {
					err = qingCloudClient.QcList(t.ResourceModel,t.Name)
				} else if t.ResourceType == 2{
					err = qingCloudClient.QcIpList(t.ResourceModel,t.Name)
				}
				if err != nil {
					errValue := fmt.Sprintf("同步青云资源失败，%v", err)
					log.Error(errValue)
					panic(errValue)
				}else {
					c <- syncStatus{
						ID: t.Id,
						Status: true,
					}
				}
			}
		}(task, ch)
	}

	return
}

// 开始同步数据
func Start() (err error) {
	if viper.GetInt(`settings.sync.cloud`) > 0 {
		td := viper.GetDuration(`settings.sync.cloud`) * time.Minute
		t := time.NewTicker(td)
		defer t.Stop()

		log.Info("Start syncing cloud resource data...")
		for {
			<-t.C
			err = syncCloud()
			if err != nil {
				log.Fatalf("同步云资产数据失败，%v", err)
				return
			}
			t.Reset(td)
		}
	}
	return
}
