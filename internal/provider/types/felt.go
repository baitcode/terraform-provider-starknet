package types

import (
	"context"
	"fmt"

	"github.com/NethermindEth/juno/core/felt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type Felt struct {
	state attr.ValueState

	*felt.Felt
}

var (
	_ basetypes.StringValuable = (*Felt)(nil)
)

func (f Felt) Type(context.Context) attr.Type {
	return FeltType{}
}

func (f Felt) Equal(o attr.Value) bool {
	other, ok := o.(Felt)

	if !ok {
		return false
	}

	if f.state != other.state {
		return false
	}

	// TODO(baitcode): check
	if f.state != attr.ValueStateKnown {
		return true
	}

	return f.Felt.Equal(other.Felt)
}

func (f Felt) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	t := FeltType{}.TerraformType(ctx)

	switch f.state {
	case attr.ValueStateKnown:
		return tftypes.NewValue(t, f.Felt), nil
	case attr.ValueStateNull:
		return tftypes.NewValue(t, nil), nil
	case attr.ValueStateUnknown:
		return tftypes.NewValue(t, tftypes.UnknownValue), nil
	default:
		panic(fmt.Sprintf("unhandled String state in ToTerraformValue: %s", f.state))
	}
}

func (f Felt) IsNull() bool {
	return f.state == attr.ValueStateNull
}

func (f Felt) IsUnknown() bool {
	return f.state == attr.ValueStateUnknown
}

func (f Felt) String() string {
	return f.Felt.String()
}

func (f Felt) ToStringValue(ctx context.Context) (basetypes.StringValue, diag.Diagnostics) {
	switch f.state {
	case attr.ValueStateKnown:
		return types.StringValue(f.String()), nil
	case attr.ValueStateNull:
		return types.StringNull(), nil
	case attr.ValueStateUnknown:
		return types.StringUnknown(), nil
	default:
		return types.StringUnknown(), diag.Diagnostics{
			diag.NewErrorDiagnostic(fmt.Sprintf("unhandled Felt state in ToStringValue: %s", f.state), ""),
		}
	}
}

func (f *Felt) FromFelt(v *felt.Felt) *Felt {
	f.Felt = v
	return f
}
