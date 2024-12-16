package types

import (
	"context"
	"fmt"

	"github.com/NethermindEth/juno/core/felt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type FeltType struct{}

var (
	_ basetypes.StringTypable = (*FeltType)(nil)

	// TODO: add validation
)

// String implements basetypes.StringTypable.
func (f FeltType) String() string {
	return "starknet.Felt"
}

// ValueType implements basetypes.StringTypable.
func (f FeltType) ValueType(context.Context) attr.Value {
	return Felt{}
}

// Equal returns true if the given type is equivalent.
func (t FeltType) Equal(o attr.Type) bool {
	other, ok := o.(FeltType)

	if !ok {
		return false
	}

	return t.String() == other.String()
}

// TerraformType implements basetypes.StringTypable.
func (f FeltType) TerraformType(ctx context.Context) tftypes.Type {
	return tftypes.String
}

// ValueFromString implements basetypes.StringTypable.
func (f FeltType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {

	if in.IsUnknown() {
		return Felt{
			state: attr.ValueStateUnknown,
		}, nil
	}

	if in.IsNull() {
		return Felt{
			state: attr.ValueStateNull,
		}, nil
	}

	value := &felt.Felt{}
	value, err := value.SetString(in.ValueString())
	if err != nil {
		diag := make(diag.Diagnostics, 1)
		diag.AddError("Failed to convert string to Felt", err.Error())
		return nil, diag
	}

	return Felt{
		Felt:  value,
		state: attr.ValueStateKnown,
	}, nil
}

// ValueFromTerraform returns a Value given a tftypes.Value.  This is meant to convert the tftypes.Value into a more convenient Go type
// for the provider to consume the data with.
func (t FeltType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	if !in.IsKnown() {
		return Felt{
			state: attr.ValueStateUnknown,
		}, nil
	}

	if in.IsNull() {
		return Felt{
			state: attr.ValueStateNull,
		}, nil
	}

	var s string
	err := in.As(&s)

	if err != nil {
		return nil, err
	}

	v := &felt.Felt{}
	v, err = v.SetString(s)
	if err != nil {
		return nil, err
	}

	return Felt{
		Felt:  v,
		state: attr.ValueStateKnown,
	}, nil
}

// ApplyTerraform5AttributePathStep implements basetypes.StringTypable.
func (f FeltType) ApplyTerraform5AttributePathStep(step tftypes.AttributePathStep) (interface{}, error) {
	return nil, fmt.Errorf("cannot apply AttributePathStep %T to %s", step, f.String())
}
