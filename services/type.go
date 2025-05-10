package services

/*eip1159交易类型，用于交易序列化*/
type Eip1559DynamicFeeTx struct {
	ChainId              string `json:"chain_id"`
	Nonce                uint64 `json:"nonce"`
	FromAddress          string `json:"from_address"`
	ToAddress            string `json:"to_address"`
	GasLimit             uint64 `json:"gas_limit"`                /*最大gas 费限制*/
	MaxFeePerGas         string `json:"max_fee_per_gas"`          /*最大 gas 每单位= baseFee + priorityFee*/
	MaxPriorityFeePerGas string `json:"max_priority_fee_per_gas"` /*每单位最大优先费*/

	// eth/erc20 amount
	Amount string `json:"amount"`
	// erc20 erc721 erc1155 contract_address
	ContractAddress string `json:"contract_address"`
}
