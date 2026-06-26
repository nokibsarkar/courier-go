package courier

import (
	"context"
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// CourierClient is the minimum portable shipment capability expected from a courier.
type CourierClient interface {
	CreateShipment(ctx context.Context, req CreateShipmentRequest) (*Shipment, error)
	TrackShipment(ctx context.Context, ref TrackingReference) (*TrackingStatus, error)
}

// BulkShipmentProvider is implemented by couriers that support bulk creation.
type BulkShipmentProvider interface {
	CreateShipments(ctx context.Context, reqs []CreateShipmentRequest) (*BulkShipmentResult, error)
}

// BalanceProvider is implemented by couriers that expose account balance.
type BalanceProvider interface {
	CurrentBalance(ctx context.Context) (*Balance, error)
}

// PaymentProvider is implemented by couriers that expose merchant payment data.
type PaymentProvider interface {
	ListPayments(ctx context.Context, opts ListOptions) (*PaymentPage, error)
	GetPayment(ctx context.Context, paymentID string) (*PaymentDetails, error)
}

// LocationProvider is implemented by couriers that expose service areas.
type LocationProvider interface {
	ListServiceAreas(ctx context.Context, opts ListOptions) (*ServiceAreaPage, error)
}

// ReturnProvider is implemented by couriers that expose return lookup/listing.
type ReturnProvider interface {
	CreateReturn(ctx context.Context, req CreateReturnRequest) (*ReturnRequest, error)
	GetReturn(ctx context.Context, returnID string) (*ReturnRequest, error)
	ListReturns(ctx context.Context, opts ListOptions) (*ReturnPage, error)
}

func AsBalanceProvider(client any) (BalanceProvider, bool) {
	provider, ok := client.(BalanceProvider)
	return provider, ok
}

func AsBulkShipmentProvider(client any) (BulkShipmentProvider, bool) {
	provider, ok := client.(BulkShipmentProvider)
	return provider, ok
}

func AsPaymentProvider(client any) (PaymentProvider, bool) {
	provider, ok := client.(PaymentProvider)
	return provider, ok
}

func AsLocationProvider(client any) (LocationProvider, bool) {
	provider, ok := client.(LocationProvider)
	return provider, ok
}

func AsReturnProvider(client any) (ReturnProvider, bool) {
	provider, ok := client.(ReturnProvider)
	return provider, ok
}

type Money struct {
	Value    decimal.Decimal
	Currency string
}

type DeliveryType string

const (
	DeliveryTypeHome  DeliveryType = "home"
	DeliveryTypePoint DeliveryType = "point"
)

type CreateShipmentRequest struct {
	MerchantOrderID string
	Recipient       Recipient
	CODAmount       Money
	DeliveryType    DeliveryType
	Note            string
	ItemDescription string
	TotalLot        int
	ProviderFields  map[string]any
}

type Recipient struct {
	Name             string
	Phone            string
	AlternativePhone string
	Email            string
	Address          string
}

type Shipment struct {
	Provider        string
	ConsignmentID   string
	MerchantOrderID string
	TrackingCode    string
	Status          string
	Recipient       Recipient
	CODAmount       Money
	Note            string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Raw             json.RawMessage
}

type BulkShipmentResult struct {
	Results []BulkShipmentItemResult
	Raw     json.RawMessage
}

type BulkShipmentItemResult struct {
	Shipment *Shipment
	Success  bool
	Error    string
	Raw      json.RawMessage
}

type TrackingReferenceType string

const (
	TrackingReferenceConsignmentID TrackingReferenceType = "consignment_id"
	TrackingReferenceMerchantOrder TrackingReferenceType = "merchant_order_id"
	TrackingReferenceTrackingCode  TrackingReferenceType = "tracking_code"
)

type TrackingReference struct {
	Type  TrackingReferenceType
	Value string
}

type TrackingStatus struct {
	Provider       string
	Reference      TrackingReference
	Status         string
	ProviderStatus string
	Raw            json.RawMessage
}

type CreateReturnRequest struct {
	Reference TrackingReference
	Reason    string
}

type ReturnRequest struct {
	Provider      string
	ID            string
	ConsignmentID string
	Reason        string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Raw           json.RawMessage
}

type Balance struct {
	Provider string
	Amount   Money
	Raw      json.RawMessage
}

type Payment struct {
	Provider  string
	ID        string
	Amount    Money
	CreatedAt time.Time
	UpdatedAt time.Time
	Raw       json.RawMessage
}

type PaymentDetails struct {
	Payment
	Consignments []map[string]any
}

type PaymentPage struct {
	Data []Payment
	Raw  json.RawMessage
}

type ServiceArea struct {
	Provider string
	ID       string
	Name     string
	Location string
	Raw      json.RawMessage
}

type ServiceAreaPage struct {
	Data []ServiceArea
	Raw  json.RawMessage
}

type ReturnPage struct {
	Data []ReturnRequest
	Raw  json.RawMessage
}

type ListOptions struct {
	Page       int
	PerPage    int
	Parameters map[string]string
}
