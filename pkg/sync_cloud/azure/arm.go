package azure

import (
	"context"
	"encoding/json"
	"fiy/app/cmdb/models/resource"
	orm "fiy/common/global"
	"fiy/common/log"
	"fiy/pkg/es"
	"fiy/tools"
	"fmt"
	"gorm.io/gorm/clause"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
)

type azure struct {
	Tk string `json:"tk"`
	Sk string `json:"sk"`
	Ak    string   `json:"ak"`
	SubK string `json:"subK"`
	Region []string `json:"region"`
}

func NewAzure(sk ,tk,subK,ak string, region []string) *azure{
	return &azure{
		Tk: tk,
		Sk: sk,
		Ak: ak,
		SubK: subK,
		Region: region,
	}
}

func (e *azure)ArmList(infoID int , infoName string )(err error){
	var (
		dataList  []resource.Data
		armClient *armcompute.VirtualMachinesClient
	)
	cred, err := azidentity.NewClientSecretCredential(
		tools.Strip(e.Tk),
		tools.Strip(e.Sk),
		tools.Strip(e.Ak),
		nil,
	)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}
	armClient = armcompute.NewVirtualMachinesClient(tools.Strip(e.SubK), cred, nil)
	ctx := context.Background()
	for _, r := range e.Region {
		pager := armClient.ListByLocation(r, nil)
		for {
			nextResult := pager.NextPage(ctx)
			if err := pager.Err(); err != nil {
				log.Fatalf("failed to advance page: %v", err)
			}
			if !nextResult {
				break
			}
			for _, instance := range pager.PageResponse().Value {

				var d []byte
				d, err = json.Marshal(instance.Properties)
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
				tmp["vmSize"]= *instance.Properties.HardwareProfile.VMSize
				delete(tmp, "hardwareProfile")
				tmp["name"]=*instance.Name
				tmp["location"] = *instance.Location

				tmp["publisher"]= *instance.Properties.StorageProfile.ImageReference.Publisher
				tmp["offer"]= *instance.Properties.StorageProfile.ImageReference.Offer
				tmp["sku"]= *instance.Properties.StorageProfile.ImageReference.SKU
				tmp["osType"]= *instance.Properties.StorageProfile.OSDisk.OSType
				tmp["osname"]= *instance.Properties.StorageProfile.OSDisk.Name
				tmp["createOption"]= *instance.Properties.StorageProfile.OSDisk.CreateOption
				tmp["diskSizeGB"]= *instance.Properties.StorageProfile.OSDisk.DiskSizeGB
				tmp["storageAccountType"]= *instance.Properties.StorageProfile.OSDisk.ManagedDisk.StorageAccountType
				delete(tmp, "storageProfile")
				tmp["computerName"] = *instance.Properties.OSProfile.ComputerName
				tmp["adminUsername"]= *instance.Properties.OSProfile.AdminUsername
				delete(tmp, "osProfile")
				delete(tmp, "diagnosticsProfile")
				delete(tmp, "provisioningState")
				//tmp["instancesID"] = tmp["id"]
				delete(tmp, "id")
				d, err = json.Marshal(tmp)
				if err != nil {
					log.Errorf("序列化服务器数据失败，%v", err)
					return
				}
				dataList = append(dataList, resource.Data{
					Uuid:   fmt.Sprintf("azure-arm-%s", *instance.Name),
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
		}
	}
	return
}
