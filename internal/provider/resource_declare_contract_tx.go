// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/NethermindEth/starknet.go/account"
	"github.com/NethermindEth/starknet.go/contracts"
	"github.com/NethermindEth/starknet.go/hash"
	"github.com/NethermindEth/starknet.go/rpc"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	framework_types "github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/baitcode/terraform-provider-starknet/internal/provider/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DeclareContractTx{}
var _ resource.ResourceWithImportState = &DeclareContractTx{}

func NewDeclareContractTxResource() resource.Resource {
	return &DeclareContractTx{}
}

// DeclareContractTx defines the resource implementation.
type DeclareContractTx struct {
	client        *rpc.Provider
	senderAddress types.Felt
	keyStore      *account.MemKeystore
	publicKey     string
}

// DeclareContractTxDataSource describes the resource data model.
type DeclareContractTxDataSource struct {
	Casm      framework_types.String `tfsdk:"compiled_casm"`
	File      framework_types.String `tfsdk:"compiled_class"`
	ClassHash framework_types.String `tfsdk:"class_hash"`
}

func (r *DeclareContractTx) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_declare_contract_tx"
}

func (r *DeclareContractTx) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example resource",

		Attributes: map[string]schema.Attribute{
			"compiled_casm": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Contract casm class path",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"compiled_class": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Contract file class path",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"class_hash": schema.StringAttribute{
				Required:            false,
				MarkdownDescription: "ClassHash for contract",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Computed: true,
			},
		},
	}
}

func (r *DeclareContractTx) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*ProviderData)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = data.client
	r.senderAddress.Felt = data.address
	r.publicKey = data.publicKey

	r.keyStore = data.keyStore
}

type ExecutionErrorData struct {
	ExecutionError   string `json:"execution_error"`
	TransactionIndex int    `json:"transaction_index"`
}

func (r *DeclareContractTx) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeclareContractTxDataSource

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	compiledClassString, err := os.ReadFile(data.File.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Can't read compiled class.",
			fmt.Sprintf("Unable to create contract, got error: %s", err),
		)
		return
	}

	compiledCasm, err := contracts.UnmarshalCasmClass(data.Casm.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid casm class",
			fmt.Sprintf("Unable to create contract, got error: %s", err),
		)
		return
	}

	compClassHash := hash.CompiledClassHash(*compiledCasm)

	tflog.Warn(ctx, fmt.Sprintf("created a resource %s", compClassHash))
	var class rpc.ContractClass
	err = json.Unmarshal(compiledClassString, &class)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Can't unmarshal compiled class: %s", err),
			fmt.Sprintf("Class: %s", compiledClassString),
		)
		return
	}
	classHash := hash.ClassHash(class)

	a, err := account.NewAccount(
		r.client,
		r.senderAddress.Felt,
		r.publicKey,
		r.keyStore, 1,
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Can't create account",
			fmt.Sprintf("Unable to create contract, got error: %s", err),
		)
		return
	}

	alreadyDeclared := false
	broadcastTx, err := SignAndEstimateDeclareTransaction(a, &class, classHash, compClassHash)
	if err != nil {

		if rpcErr, ok := err.(*rpc.RPCError); ok {
			if errData, ok := rpcErr.Data.(map[string]interface{}); ok && rpcErr.Code == 41 && strings.Contains(errData["execution_error"].(string), "is already declared") {
				alreadyDeclared = true
				data.ClassHash = framework_types.StringValue(classHash.String())
			}
		}
		if !alreadyDeclared {
			resp.Diagnostics.AddError(
				"Can't sign and estimate transaction hash",
				fmt.Sprintf("Failed with error: %s", err),
			)
			return
		}
	}

	if !alreadyDeclared {
		response, err := a.SendTransaction(context.Background(), broadcastTx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Can't send transaction",
				fmt.Sprintf("Unable to create contract, got error: %s", err),
			)
			return
		}

		for {
			block, err := a.WaitForTransactionReceipt(
				context.Background(),
				response.TransactionHash,
				5*time.Second,
			)
			if err != nil {
				resp.Diagnostics.AddError(
					"Transaction failed",
					fmt.Sprintf("Unable to create contract, got error: %s", err),
				)
				return
			}
			if block.FinalityStatus == rpc.TxnFinalityStatusAcceptedOnL2 {
				break
			}
		}
	}

	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeclareContractTx) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeclareContractTxDataSource

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeclareContractTx) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DeclareContractTxDataSource

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeclareContractTx) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeclareContractTxDataSource

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }
}

func (r *DeclareContractTx) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
