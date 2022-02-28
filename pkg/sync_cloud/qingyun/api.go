package qingyun

import (
	"github.com/yunify/qingcloud-sdk-go/config"
	"github.com/yunify/qingcloud-sdk-go/logger"
	"github.com/yunify/qingcloud-sdk-go/request"
	"github.com/yunify/qingcloud-sdk-go/request/data"
	"github.com/yunify/qingcloud-sdk-go/request/errors"
)
type QingCloudServicePropertiesS struct {
}
type QingCloudServiceS struct {
	Config     *config.Config
	Properties *QingCloudServicePropertiesS
}
type AssociateEIPInput struct {
	EIP      *string `json:"eip" name:"eip" location:"params"`           // Required
	Instance *string `json:"instance" name:"instance" location:"params"` // Required
	Nic  *string `json:"nic" name:"nic" location:"params"`
}
type EIPServiceProperties struct {
	// QingCloud Zone ID
	Zone *string `json:"zone" name:"zone"` // Required
}
type EIPServices struct {
	Config     *config.Config
	Properties *EIPServiceProperties
}
type AssociateEIPOutput struct {
	Message *string `json:"message" name:"message"`
	Action  *string `json:"action" name:"action" location:"elements"`
	JobID   *string `json:"job_id" name:"job_id" location:"elements"`
	RetCode *int    `json:"ret_code" name:"ret_code" location:"elements"`
}
func (s *QingCloudServiceS) EIPS(zone string) (*EIPServices, error) {
	properties := &EIPServiceProperties{
		Zone: &zone,
	}

	return &EIPServices{Config: s.Config, Properties: properties}, nil
}
func Init(c *config.Config) (*QingCloudServiceS, error) {
	properties := &QingCloudServicePropertiesS{}
	logger.SetLevel(c.LogLevel)
	return &QingCloudServiceS{Config: c, Properties: properties}, nil
}
func  (s EIPServices)AssociateEIPss(i *AssociateEIPInput) (*AssociateEIPOutput, error) {
	if i == nil {
		i = &AssociateEIPInput{}
	}
	o := &data.Operation{
		Config:        s.Config,
		Properties:    s.Properties,
		APIName:       "AssociateEip",
		RequestMethod: "GET",
	}

	x := &AssociateEIPOutput{}
	r, err := request.New(o, i, x)

	if err != nil {
		return nil, err
	}

	err = r.Send()
	if err != nil {
		return nil, err
	}

	return x, err
}
func (v *AssociateEIPInput) Validate() error {

	if v.EIP == nil {
		return errors.ParameterRequiredError{
			ParameterName: "EIP",
			ParentName:    "AssociateEIPInput",
		}
	}

	if v.Instance == nil {
		return errors.ParameterRequiredError{
			ParameterName: "Instance",
			ParentName:    "AssociateEIPInput",
		}
	}
	if v.Nic == nil {
		return errors.ParameterRequiredError{
			ParameterName: "Nic",
			ParentName:    "AssociateEIPInput",
		}
	}

	return nil
}
