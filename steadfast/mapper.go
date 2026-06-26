package steadfast

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	courier "github.com/nokibsarkar/courier-go"
	"github.com/shopspring/decimal"
)

var invoicePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func buildOrderRequest(req courier.CreateShipmentRequest) (orderRequest, error) {
	invoice := strings.TrimSpace(req.MerchantOrderID)
	if invoice == "" {
		return orderRequest{}, &courier.Error{Kind: courier.ErrorKindValidation, Field: "merchant_order_id", Message: "merchant order id is required"}
	}
	if !invoicePattern.MatchString(invoice) {
		return orderRequest{}, &courier.Error{Kind: courier.ErrorKindValidation, Field: "merchant_order_id", Message: "must contain only letters, numbers, hyphens, and underscores"}
	}
	name := strings.TrimSpace(req.Recipient.Name)
	if name == "" {
		return orderRequest{}, &courier.Error{Kind: courier.ErrorKindValidation, Field: "recipient.name", Message: "recipient name is required"}
	}
	if len(name) > 100 {
		return orderRequest{}, &courier.Error{Kind: courier.ErrorKindValidation, Field: "recipient.name", Message: "recipient name cannot exceed 100 characters"}
	}
	phone, err := normalizePhone(req.Recipient.Phone, "recipient.phone")
	if err != nil {
		return orderRequest{}, err
	}
	address := strings.TrimSpace(req.Recipient.Address)
	if address == "" {
		return orderRequest{}, &courier.Error{Kind: courier.ErrorKindValidation, Field: "recipient.address", Message: "recipient address is required"}
	}
	if len(address) > 250 {
		return orderRequest{}, &courier.Error{Kind: courier.ErrorKindValidation, Field: "recipient.address", Message: "recipient address cannot exceed 250 characters"}
	}
	if req.CODAmount.Value.IsNegative() {
		return orderRequest{}, &courier.Error{Kind: courier.ErrorKindValidation, Field: "cod_amount", Message: "cod amount cannot be negative"}
	}
	deliveryType, err := mapDeliveryType(req.DeliveryType)
	if err != nil {
		return orderRequest{}, err
	}

	payload := orderRequest{
		Invoice:          invoice,
		RecipientName:    name,
		RecipientPhone:   phone,
		RecipientAddress: address,
		CODAmount:        providerAmountFromMoney(req.CODAmount),
		DeliveryType:     deliveryType,
		Note:             strings.TrimSpace(req.Note),
		ItemDescription:  strings.TrimSpace(req.ItemDescription),
		TotalLot:         req.TotalLot,
	}
	if req.Recipient.AlternativePhone != "" {
		alt, err := normalizePhone(req.Recipient.AlternativePhone, "recipient.alternative_phone")
		if err != nil {
			return orderRequest{}, err
		}
		payload.AlternativePhone = alt
	}
	if req.Recipient.Email != "" {
		email := strings.TrimSpace(req.Recipient.Email)
		if !strings.Contains(email, "@") {
			return orderRequest{}, &courier.Error{Kind: courier.ErrorKindValidation, Field: "recipient.email", Message: "invalid email address"}
		}
		payload.RecipientEmail = email
	}
	return payload, nil
}

func normalizePhone(value, field string) (string, error) {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	phone := b.String()
	if len(phone) != 11 {
		return "", &courier.Error{Kind: courier.ErrorKindValidation, Field: field, Message: "phone must contain exactly 11 digits"}
	}
	return phone, nil
}

func mapDeliveryType(value courier.DeliveryType) (int, error) {
	switch value {
	case "", courier.DeliveryTypeHome:
		return 0, nil
	case courier.DeliveryTypePoint:
		return 1, nil
	default:
		return 0, &courier.Error{Kind: courier.ErrorKindValidation, Field: "delivery_type", Message: "delivery type must be home or point"}
	}
}

func providerAmountFromMoney(m courier.Money) float64 {
	value, _ := m.Value.Float64()
	return value
}

func moneyFromProviderAmount(amount any) courier.Money {
	switch v := amount.(type) {
	case float64:
		return courier.Money{Value: decimal.NewFromFloat(v), Currency: "BDT"}
	case float32:
		return courier.Money{Value: decimal.NewFromFloat32(v), Currency: "BDT"}
	case int:
		return courier.Money{Value: decimal.NewFromInt(int64(v)), Currency: "BDT"}
	case int64:
		return courier.Money{Value: decimal.NewFromInt(v), Currency: "BDT"}
	case decimal.Decimal:
		return courier.Money{Value: v, Currency: "BDT"}
	case string:
		parsed, err := decimal.NewFromString(strings.TrimSpace(v))
		if err == nil {
			return courier.Money{Value: parsed, Currency: "BDT"}
		}
	}
	return courier.Money{Currency: "BDT"}
}

func parseProviderTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func mapOrderResponse(out orderResponse, raw json.RawMessage) *courier.Shipment {
	return &courier.Shipment{
		Provider:        ProviderName,
		ConsignmentID:   stringify(out.ConsignmentID),
		MerchantOrderID: out.Invoice,
		TrackingCode:    out.TrackingCode,
		Status:          out.Status,
		Recipient: courier.Recipient{
			Name:    out.RecipientName,
			Phone:   out.RecipientPhone,
			Address: out.RecipientAddress,
		},
		CODAmount: moneyFromProviderAmount(out.CODAmount),
		Note:      out.Note,
		CreatedAt: parseProviderTime(out.CreatedAt),
		UpdatedAt: parseProviderTime(out.UpdatedAt),
		Raw:       raw,
	}
}

func trackingPath(ref courier.TrackingReference) (string, error) {
	value := strings.TrimSpace(ref.Value)
	if value == "" {
		return "", &courier.Error{Kind: courier.ErrorKindValidation, Field: "reference.value", Message: "reference value is required"}
	}
	switch ref.Type {
	case courier.TrackingReferenceConsignmentID:
		if _, err := strconv.Atoi(value); err != nil {
			return "", &courier.Error{Kind: courier.ErrorKindValidation, Field: "reference.value", Message: "consignment id must be numeric", Err: err}
		}
		return "/status_by_cid/" + url.PathEscape(value), nil
	case courier.TrackingReferenceMerchantOrder:
		if !invoicePattern.MatchString(value) {
			return "", &courier.Error{Kind: courier.ErrorKindValidation, Field: "reference.value", Message: "merchant order id must contain only letters, numbers, hyphens, and underscores"}
		}
		return "/status_by_invoice/" + url.PathEscape(value), nil
	case courier.TrackingReferenceTrackingCode:
		return "/status_by_trackingcode/" + url.PathEscape(value), nil
	default:
		return "", &courier.Error{Kind: courier.ErrorKindValidation, Field: "reference.type", Message: "unsupported tracking reference type"}
	}
}

func normalizeStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "delivered", "completed":
		return "delivered"
	case "cancelled", "canceled":
		return "cancelled"
	case "pending":
		return "pending"
	case "in_review", "processing", "in transit", "in_transit":
		return "in_transit"
	case "returned", "return":
		return "returned"
	default:
		if status == "" {
			return "unknown"
		}
		return strings.ToLower(strings.ReplaceAll(status, " ", "_"))
	}
}

func stringify(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return fmt.Sprint(v)
	}
}
