package steadfast

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	courier "github.com/nokibsarkar/courier-go"
	"github.com/shopspring/decimal"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCreateShipment(t *testing.T) {
	var gotPath string
	var gotHeaders http.Header
	var gotPayload orderRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotHeaders = r.Header.Clone()
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(orderEnvelopeResponse{
			Status:  200,
			Message: "Consignment has been created successfully.",
			Consignment: orderResponse{
				ConsignmentID:    123,
				Invoice:          gotPayload.Invoice,
				TrackingCode:     "TRK123",
				RecipientName:    gotPayload.RecipientName,
				RecipientPhone:   gotPayload.RecipientPhone,
				RecipientAddress: gotPayload.RecipientAddress,
				CODAmount:        gotPayload.CODAmount,
				Status:           "pending",
				CreatedAt:        "2021-03-21T07:05:31.000000Z",
				UpdatedAt:        "2021-03-21T07:06:31.000000Z",
			},
		})
	}))
	defer server.Close()

	client, err := New(Config{APIKey: "api", SecretKey: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	shipment, err := client.CreateShipment(context.Background(), courier.CreateShipmentRequest{
		MerchantOrderID: "ORD-1",
		Recipient: courier.Recipient{
			Name:    "Noki",
			Phone:   "01711-111111",
			Address: "Dhaka",
		},
		CODAmount:    courier.Money{Value: decimal.RequireFromString("1250.50"), Currency: "BDT"},
		DeliveryType: courier.DeliveryTypeHome,
	})
	if err != nil {
		t.Fatalf("create shipment: %v", err)
	}

	if gotPath != "/create_order" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotHeaders.Get("Api-Key") != "api" || gotHeaders.Get("Secret-Key") != "secret" {
		t.Fatalf("auth headers not set: %#v", gotHeaders)
	}
	if gotPayload.CODAmount != 1250.50 {
		t.Fatalf("cod amount = %v", gotPayload.CODAmount)
	}
	if gotPayload.RecipientPhone != "01711111111" {
		t.Fatalf("phone = %q", gotPayload.RecipientPhone)
	}
	if shipment.ConsignmentID != "123" || shipment.TrackingCode != "TRK123" {
		t.Fatalf("unexpected shipment: %#v", shipment)
	}
	if !shipment.CreatedAt.Equal(time.Date(2021, 3, 21, 7, 5, 31, 0, time.UTC)) {
		t.Fatalf("created at = %s", shipment.CreatedAt)
	}
	if !shipment.CODAmount.Value.Equal(decimal.RequireFromString("1250.50")) {
		t.Fatalf("shipment cod amount = %s", shipment.CODAmount.Value)
	}
}

func TestNewReadsEnvironmentVariables(t *testing.T) {
	t.Setenv("STEADFAST_API_KEY", "env-api")
	t.Setenv("STEADFAST_SECRET_KEY", "env-secret")
	t.Setenv("STEADFAST_BASE_URL", "https://tenant.example.test/api/v1/")

	client, err := New(Config{})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if client.apiKey != "env-api" {
		t.Fatalf("api key = %q", client.apiKey)
	}
	if client.secretKey != "env-secret" {
		t.Fatalf("secret key = %q", client.secretKey)
	}
	if client.baseURL != "https://tenant.example.test/api/v1" {
		t.Fatalf("base url = %q", client.baseURL)
	}
}

func TestExplicitConfigOverridesEnvironmentVariables(t *testing.T) {
	t.Setenv("STEADFAST_API_KEY", "env-api")
	t.Setenv("STEADFAST_SECRET_KEY", "env-secret")
	t.Setenv("STEADFAST_BASE_URL", "https://env.example.test")

	client, err := New(Config{
		APIKey:    "tenant-api",
		SecretKey: "tenant-secret",
		BaseURL:   "https://tenant.example.test/api/v1",
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if client.apiKey != "tenant-api" || client.secretKey != "tenant-secret" {
		t.Fatalf("explicit credentials were not used: %#v", client)
	}
	if client.baseURL != "https://tenant.example.test/api/v1" {
		t.Fatalf("base url = %q", client.baseURL)
	}
}

func TestInjectedHTTPClientAllowsTenantScopedClients(t *testing.T) {
	var seen []string
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			seen = append(seen, req.Header.Get("Api-Key")+":"+req.URL.Host+req.URL.Path)
			body := `{"status":200,"current_balance":"120.50"}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewBufferString(body)),
				Request:    req,
			}, nil
		}),
	}

	tenantA, err := New(Config{
		APIKey:     "tenant-a",
		SecretKey:  "secret-a",
		BaseURL:    "https://a.example.test/api/v1",
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("new tenant a: %v", err)
	}
	tenantB, err := New(Config{
		APIKey:     "tenant-b",
		SecretKey:  "secret-b",
		BaseURL:    "https://b.example.test/api/v1",
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("new tenant b: %v", err)
	}

	if _, err := tenantA.CurrentBalance(context.Background()); err != nil {
		t.Fatalf("tenant a balance: %v", err)
	}
	if _, err := tenantB.CurrentBalance(context.Background()); err != nil {
		t.Fatalf("tenant b balance: %v", err)
	}

	want := []string{
		"tenant-a:a.example.test/api/v1/get_balance",
		"tenant-b:b.example.test/api/v1/get_balance",
	}
	if len(seen) != len(want) {
		t.Fatalf("seen = %#v", seen)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("seen[%d] = %q, want %q", i, seen[i], want[i])
		}
	}
}

func TestTrackShipmentByMerchantOrder(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(statusResponse{Status: 200, DeliveryStatus: "Delivered"})
	}))
	defer server.Close()

	client, err := New(Config{APIKey: "api", SecretKey: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	status, err := client.TrackShipment(context.Background(), courier.TrackingReference{
		Type:  courier.TrackingReferenceMerchantOrder,
		Value: "ORD-1",
	})
	if err != nil {
		t.Fatalf("track shipment: %v", err)
	}
	if gotPath != "/status_by_invoice/ORD-1" {
		t.Fatalf("path = %q", gotPath)
	}
	if status.Status != "delivered" || status.ProviderStatus != "Delivered" {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestCreateShipmentValidation(t *testing.T) {
	client, err := New(Config{APIKey: "api", SecretKey: "secret"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.CreateShipment(context.Background(), courier.CreateShipmentRequest{
		MerchantOrderID: "ORD@1",
		Recipient: courier.Recipient{
			Name:    "Noki",
			Phone:   "01711111111",
			Address: "Dhaka",
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !courier.IsKind(err, courier.ErrorKindValidation) {
		t.Fatalf("expected validation error, got %T %v", err, err)
	}
}

func TestCreateShipmentsRejectsMoreThan500(t *testing.T) {
	client, err := New(Config{APIKey: "api", SecretKey: "secret"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	reqs := make([]courier.CreateShipmentRequest, 501)
	_, err = client.CreateShipments(context.Background(), reqs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !courier.IsKind(err, courier.ErrorKindValidation) {
		t.Fatalf("expected validation error, got %T %v", err, err)
	}
}

func TestCreateShipmentsUsesSteadfastBulkEndpointAndPayload(t *testing.T) {
	var gotPath string
	var gotPayload bulkOrderRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode([]bulkOrderItemResponse{
			{
				Invoice:          "ORD-1",
				RecipientName:    "Noki",
				RecipientAddress: "Dhaka",
				RecipientPhone:   "01711111111",
				CODAmount:        "1000.00",
				ConsignmentID:    123,
				TrackingCode:     "TRK123",
				Status:           "success",
			},
		})
	}))
	defer server.Close()

	client, err := New(Config{APIKey: "api", SecretKey: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	result, err := client.CreateShipments(context.Background(), []courier.CreateShipmentRequest{
		{
			MerchantOrderID: "ORD-1",
			Recipient: courier.Recipient{
				Name:    "Noki",
				Phone:   "01711111111",
				Address: "Dhaka",
			},
			CODAmount: courier.Money{Value: decimal.NewFromInt(1000), Currency: "BDT"},
		},
	})
	if err != nil {
		t.Fatalf("create shipments: %v", err)
	}
	if gotPath != "/create_order/bulk-order" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotPayload.Data == "" {
		t.Fatal("expected data payload")
	}
	var orders []orderRequest
	if err := json.Unmarshal([]byte(gotPayload.Data), &orders); err != nil {
		t.Fatalf("bulk data was not json encoded orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Invoice != "ORD-1" {
		t.Fatalf("unexpected orders: %#v", orders)
	}
	if len(result.Results) != 1 || !result.Results[0].Success {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestHTTPErrorClassification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"bad credentials"}`))
	}))
	defer server.Close()

	client, err := New(Config{APIKey: "api", SecretKey: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.CurrentBalance(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !courier.IsKind(err, courier.ErrorKindAuthentication) {
		t.Fatalf("expected authentication error, got %T %v", err, err)
	}
}
