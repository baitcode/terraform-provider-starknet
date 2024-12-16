// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/hex"
	"math/big"
	"os"

	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/starknet.go/account"
	"github.com/NethermindEth/starknet.go/rpc"

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
	PrivateKeyPath types.String `tfsdk:"private_key_path"`
	PublicKeyPath  types.String `tfsdk:"public_key_path"`
	RpcEndpoint    types.String `tfsdk:"rpc_endpoint"`
}

type ProviderData struct {
	client   *rpc.Provider
	keyStore *account.MemKeystore
	address  *felt.Felt
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
	client, err := rpc.NewProvider(data.RpcEndpoint.String())
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
	skData, err := os.ReadFile(data.PrivateKeyPath.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read private key file",
			err.Error(),
		)
		return
	}

	pkData, err := os.ReadFile(data.PublicKeyPath.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read public key file",
			err.Error(),
		)
		return
	}
	pk := hex.EncodeToString(pkData)

	ks := account.NewMemKeystore()
	privKeyBI := new(big.Int).SetBytes(skData)
	ks.Put(pk, privKeyBI)

	address := &felt.Felt{}
	address = address.SetBytes(pkData)

	providerData := &ProviderData{
		client:   client,
		keyStore: ks,
		address:  address,
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *StarknetProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewExampleResource,
	}
}

func (p *StarknetProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAccountDataSource,
	}
}

func (p *StarknetProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		NewExampleFunction,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &StarknetProvider{
			version: version,
		}
	}
}
