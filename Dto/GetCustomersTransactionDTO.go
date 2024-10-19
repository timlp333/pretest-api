package Dto

type GetCustomersTransactionDTO struct {
	BaseRespDTO
	Data CustomerData `json:"data"`
}
type CustomerTransactionData struct {
	BaseDataDTO
	Items []map[string]interface{} `json:"items"`
}
