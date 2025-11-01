package models

type (
	Request struct {
		Url string `json:"url"`
	}
	Response struct {
		Result string `json:"result"`
	}
)
