package models

type (
	Request struct {
		URL string `json:"url"`
	}
	Response struct {
		Result string `json:"result"`
	}
	BatchRequest struct {
		CorrelationID string `json:"correlation_id"`
		OriginalUrl   string `json:"original_url"`
	}
	BatchResponse struct {
		CorrelationID string `json:"correlation_id"`
		ShortUrl      string `json:"short_url"`
	}
)
