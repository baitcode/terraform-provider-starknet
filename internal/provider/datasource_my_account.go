// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/NethermindEth/starknet.go/rpc"

	"github.com/baitcode/terraform-provider-starknet/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &MyAccountDataSource{}

func NewMyAccountDataSource() datasource.DataSource {
	return &MyAccountDataSource{}
}

// MyAccountDataSource defines the data source implementation.
type MyAccountDataSource struct {
	client  *rpc.Provider
	address types.Felt
}

// AccountDataSourceModel describes the data source data model.
type AccountDataSourceModel struct {
	Address   types.Felt `tfsdk:"address"`
	ClassHash types.Felt `tfsdk:"class_hash"`
	// PublicKey framework_types.String `tfsdk:"public_key"`
}

func (d *MyAccountDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_my_account"
}

func (d *MyAccountDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Starknet account data source",

		Attributes: map[string]schema.Attribute{
			"address": schema.StringAttribute{
				CustomType:          types.FeltType{},
				MarkdownDescription: "Account address",
				Required:            false,
				Computed:            true,
			},
			"class_hash": schema.StringAttribute{
				CustomType:          types.FeltType{},
				MarkdownDescription: "Class hash",
				Required:            false,
				Computed:            true,
			},
			// "public_key": schema.StringAttribute{
			// 	CustomType:          framework_types.StringType,
			// 	MarkdownDescription: "Public Key",
			// 	Required:            false,
			// 	Computed:            true,
			// },
		},
	}
}

func (d *MyAccountDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*ProviderData)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = data.client
	// TODO: precedence needed. only override if not set
	d.address.Felt = data.address
	// d.publicKey = data.publicKey
}

func (d *MyAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	address := d.address.Felt

	classHash, err := d.client.ClassHashAt(
		context.Background(),
		rpc.WithBlockTag("latest"),
		address,
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Error reading account %s class hash: %s", address.String(), err),
		)
		return
	}

	data.Address = d.address
	data.ClassHash.Felt = classHash

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
