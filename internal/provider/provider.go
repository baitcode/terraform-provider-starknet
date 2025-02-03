// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"math/big"
	"os"
	"strings"

	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/starknet.go/account"
	"github.com/NethermindEth/starknet.go/rpc"
	"github.com/NethermindEth/starknet.go/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &StarknetProvider{}
var _ provider.ProviderWithFunctions = &StarknetProvider{}

// StarknetProvider defines the provider implementation.
type StarknetProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// StarknetProviderModel describes the provider data model.
type StarknetProviderModel struct {
	ChainId        types.String `tfsdk:"chain_id"`
	Address        types.String `tfsdk:"address"`
	PrivateKeyPath types.String `tfsdk:"private_key_path"`
	PublicKeyPath  types.String `tfsdk:"public_key_path"`
	RpcEndpoint    types.String `tfsdk:"rpc_endpoint"`
}

type ProviderData struct {
	client    *rpc.Provider
	keyStore  *account.MemKeystore
	address   *felt.Felt
	publicKey string
}

func (p *StarknetProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "starknet"
	resp.Version = p.version
}

func (p *StarknetProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			// TODO: Maybe not neeeded
			"chain_id": schema.StringAttribute{
				MarkdownDescription: "Starknet chain identifier.",
				Required:            true,
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "Admin account address.",
				Required:            true,
			},
			"private_key_path": schema.StringAttribute{
				MarkdownDescription: "Admin account secret key file path.",
				Required:            true,
			},
			"public_key_path": schema.StringAttribute{
				MarkdownDescription: "Admin account public key file path.",
				Required:            true,
			},
			"rpc_endpoint": schema.StringAttribute{
				MarkdownDescription: "Node API endpoint.",
				Required:            true,
			},
		},
	}
}

func (p *StarknetProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data StarknetProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create RPC client
	client, err := rpc.NewProvider(data.RpcEndpoint.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create Starknet provider",
			err.Error(),
		)
		return
	}

	// Check ChainID
	chain_id, err := client.ChainID(context.Background())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to obtain ChainId from rpc endpoint",
			err.Error(),
		)
		return
	}

	if chain_id != data.ChainId.String() {
		resp.Diagnostics.AddWarning(
			"ChainId mismatch",
			"ChainId from rpc endpoint does not match the one provided in config."+
				"Overriding the config value with the one from rpc endpoint.",
		)
	}

	// Load keys
	publicKeyData, err := os.ReadFile(data.PublicKeyPath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read public key file",
			err.Error(),
		)
		return
	}
	publicKey := strings.TrimSpace(string(publicKeyData))

	secretKeyData, err := os.ReadFile(data.PrivateKeyPath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read private key file",
			err.Error(),
		)
		return
	}
	secretKey := strings.TrimSpace(string(secretKeyData))

	privateKeyInt, ok := new(big.Int).SetString(secretKey, 0)
	if !ok {
		resp.Diagnostics.AddError(
			"Failed",
			"Failed to convert secret key string to big.Int",
		)
	}

	address := data.Address.ValueString()
	ks := account.NewMemKeystore()
	ks.Put(publicKey, privateKeyInt)

	addressFelt, err := utils.HexToFelt(address)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid address",
			err.Error(),
		)
		return
	}

	providerData := &ProviderData{
		client:    client,
		keyStore:  ks,
		address:   addressFelt,
		publicKey: publicKey,
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *StarknetProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDeclareContractTxResource,
	}
}

func (p *StarknetProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewMyAccountDataSource,
	}
}

func (p *StarknetProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		NewRandomSaltFunction,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &StarknetProvider{
			version: version,
		}
	}
}
