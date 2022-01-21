package tencent
import(
	"encoding/json"
	"fiy/app/cmdb/models/resource"
	orm "fiy/common/global"
	"fiy/common/log"
	"fiy/tools"
	"fmt"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm/clause"
)
type tenCetYun struct {
	SK     string   `json:"sk"`
	AK     string   `json:"ak"`
	Region []string `json:"region"`
}

func NewTencentYun(sk,ak string,region []string) *tenCetYun{
	return &tenCetYun{
		SK: sk,
		AK: ak,
		Region: region,
	}
}

func (c *tenCetYun) TccList(infoID int)(err error) {
	var (
		response  *v20170312.DescribeInstancesResponse
		cvmList   []*v20170312.Instance
		dataList  []resource.Data
		cvmClient *v20170312.Client
	)
	for _, s := range c.Region {
		credential :=common.NewCredential(tools.Strip(c.AK),tools.Strip(c.SK))

		cvmClient, err = v20170312.NewClient(
			credential,
			tools.Strip(s),
			profile.NewClientProfile(),
		)
		if err != nil {
			log.Errorf("创建客户端连接失败，%v", err)
			return
		}
		request := v20170312.NewDescribeInstancesRequest()
		response, err = cvmClient.DescribeInstances(request)
		if err != nil {
			log.Errorf("查询ECS实例列表失败，%v", err)
			return
		}
	}
		cvmList = append(cvmList, response.Response.InstanceSet...)
		for _, instance := range cvmList {
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
				Uuid:   fmt.Sprintf("tencentyun-cvm-%s", instance.InstanceId),
				InfoId: infoID,
				Status: 0,
				Data:   d,
			})

		}
		err = orm.Eloquent.Model(&resource.Data{}).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "uuid"}},
			DoUpdates: clause.AssignmentColumns([]string{"data"}),
		}).Create(&dataList).Error
		return

}
