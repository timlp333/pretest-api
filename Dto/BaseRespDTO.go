package Dto

type BaseRespDTO struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type BaseDataDTO struct {
	Total int `json:"total"`
}
