package provider

import (
	"context"
	"strconv"

	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/starknet.go/account"
	"github.com/NethermindEth/starknet.go/rpc"
	"github.com/NethermindEth/starknet.go/utils"
)

func GetFeeForDeclareV2(
	a *account.Account,
	class *rpc.ContractClass,
	classHash *felt.Felt,
	compiledClassHash *felt.Felt,
) (*uint64, error) {
	nonce, err := a.Nonce(
		context.Background(),
		rpc.BlockID{Tag: "latest"},
		a.AccountAddress,
	)
	if err != nil {
		return nil, err
	}

	tx := rpc.DeclareTxnV2{
		SenderAddress:     a.AccountAddress,
		Type:              rpc.TransactionType_Declare,
		Version:           rpc.TransactionV2,
		ClassHash:         classHash,
		CompiledClassHash: compiledClassHash,
		Nonce:             nonce,
		MaxFee:            utils.Uint64ToFelt(0),
	}

	err = a.SignDeclareTransaction(context.Background(), &tx)
	if err != nil {
		return nil, err
	}

	broadcastTxForEstimation := rpc.BroadcastDeclareTxnV2{
		Nonce:             tx.Nonce,
		MaxFee:            tx.MaxFee,
		Type:              tx.Type,
		Version:           tx.Version,
		Signature:         tx.Signature,
		SenderAddress:     tx.SenderAddress,
		CompiledClassHash: tx.CompiledClassHash,
		ContractClass:     *class,
	}

	estimation, err := a.EstimateFee(
		context.Background(),
		[]rpc.BroadcastTxn{broadcastTxForEstimation},
		[]rpc.SimulationFlag{},
		rpc.WithBlockTag("latest"),
	)
	if err, ok := err.(*rpc.RPCError); ok {
		return nil, err
	}

	var fee uint64 = 1
	if len(estimation) == 0 {
		return &fee, nil
	}

	fee, err = strconv.ParseUint(estimation[0].OverallFee.String(), 0, 64)
	if err != nil {
		return nil, err
	}

	return &fee, nil
}

func SignAndEstimateDeclareTransaction(
	a *account.Account,
	class *rpc.ContractClass,
	classHash *felt.Felt,
	compiledClassHash *felt.Felt,
) (*rpc.BroadcastDeclareTxnV2, error) {
	fee, err := GetFeeForDeclareV2(a, class, classHash, compiledClassHash)
	if err != nil {
		return nil, err
	}

	nonce, err := a.Nonce(
		context.Background(),
		rpc.BlockID{Tag: "latest"},
		a.AccountAddress,
	)

	if err != nil {
		return nil, err
	}

	tx := rpc.DeclareTxnV2{
		SenderAddress:     a.AccountAddress,
		Type:              rpc.TransactionType_Declare,
		Version:           rpc.TransactionV2,
		ClassHash:         classHash,
		CompiledClassHash: compiledClassHash,
		Nonce:             nonce,
		MaxFee:            utils.Uint64ToFelt(*fee),
	}

	err = a.SignDeclareTransaction(context.Background(), &tx)
	if err != nil {
		return nil, err
	}

	broadcastTx := rpc.BroadcastDeclareTxnV2{
		Nonce:             tx.Nonce,
		MaxFee:            tx.MaxFee,
		Type:              tx.Type,
		Version:           tx.Version,
		Signature:         tx.Signature,
		SenderAddress:     tx.SenderAddress,
		CompiledClassHash: tx.CompiledClassHash,
		ContractClass:     *class,
	}

	return &broadcastTx, nil
}
