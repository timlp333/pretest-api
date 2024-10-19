package Dto

type GetCustomersDTO struct {
	BaseRespDTO
	Data CustomerData `json:"data"`
}
type CustomerData struct {
	BaseDataDTO
	Items []map[string]interface{} `json:"items"`
}
