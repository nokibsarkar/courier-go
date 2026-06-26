package steadfast

type orderRequest struct {
	Invoice          string  `json:"invoice"`
	RecipientName    string  `json:"recipient_name"`
	RecipientPhone   string  `json:"recipient_phone"`
	RecipientAddress string  `json:"recipient_address"`
	CODAmount        float64 `json:"cod_amount"`
	DeliveryType     int     `json:"delivery_type"`
	AlternativePhone string  `json:"alternative_phone,omitempty"`
	RecipientEmail   string  `json:"recipient_email,omitempty"`
	Note             string  `json:"note,omitempty"`
	ItemDescription  string  `json:"item_description,omitempty"`
	TotalLot         int     `json:"total_lot,omitempty"`
}

type orderEnvelopeResponse struct {
	Status      int           `json:"status"`
	Message     string        `json:"message"`
	Consignment orderResponse `json:"consignment"`
}

type orderResponse struct {
	ConsignmentID    any    `json:"consignment_id"`
	Invoice          string `json:"invoice"`
	TrackingCode     string `json:"tracking_code"`
	RecipientName    string `json:"recipient_name"`
	RecipientPhone   string `json:"recipient_phone"`
	RecipientAddress string `json:"recipient_address"`
	CODAmount        any    `json:"cod_amount"`
	Status           string `json:"status"`
	Note             string `json:"note"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type bulkOrderRequest struct {
	Data string `json:"data"`
}

type bulkOrderItemResponse struct {
	Invoice          string `json:"invoice"`
	RecipientName    string `json:"recipient_name"`
	RecipientAddress string `json:"recipient_address"`
	RecipientPhone   string `json:"recipient_phone"`
	CODAmount        any    `json:"cod_amount"`
	Note             string `json:"note"`
	ConsignmentID    any    `json:"consignment_id"`
	TrackingCode     string `json:"tracking_code"`
	Status           string `json:"status"`
	Error            string `json:"error"`
}

type statusResponse struct {
	Status         int    `json:"status"`
	DeliveryStatus string `json:"delivery_status"`
}

type balanceResponse struct {
	Status         int `json:"status"`
	CurrentBalance any `json:"current_balance"`
}
