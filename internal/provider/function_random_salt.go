// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_       function.Function = RandomSaltFunction{}
	MaxSalt *big.Int          = big.NewInt(0).Lsh(big.NewInt(1), 252)
)
var x types.Number

func NewRandomSaltFunction() function.Function {
	return RandomSaltFunction{}
}

type RandomSaltFunction struct{}

func (r RandomSaltFunction) Metadata(_ context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "random_salt"
}

func (r RandomSaltFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Random Salt generator",
		MarkdownDescription: "Generates 252 random bits",
		Parameters:          []function.Parameter{},
		Return:              function.NumberReturn{},
	}
}

func (r RandomSaltFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {

	data, err := rand.Int(rand.Reader, MaxSalt)
	if err != nil {
		function.ConcatFuncErrors(
			function.NewFuncError(
				fmt.Sprintf("failed to generate random salt: %s", err.Error()),
			),
		)
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Result.Set(ctx, data))
}
