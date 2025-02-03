package provider

import "github.com/hashicorp/terraform-plugin-framework/resource"

var _ resource.Resource = &DeployContractTx{}
var _ resource.ResourceWithImportState = &DeployContractTx{}

func NewDeployContractTxResource() resource.Resource {
	return &DeclareContractTx{}
}
