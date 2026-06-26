package steadfast

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	courier "github.com/nokibsarkar/courier-go"
)

const (
	ProviderName = "steadfast"
	DefaultURL   = "https://portal.packzy.com/api/v1"
)

type Config struct {
	APIKey    string
	SecretKey string
	BaseURL   string

	HTTPClient *http.Client

	MaxRetries int
	RetryWait  time.Duration

	APIKeyHeader    string
	SecretKeyHeader string
}

type Client struct {
	apiKey          string
	secretKey       string
	baseURL         string
	httpClient      *http.Client
	maxRetries      int
	retryWait       time.Duration
	apiKeyHeader    string
	secretKeyHeader string
}

var (
	_ courier.CourierClient        = (*Client)(nil)
	_ courier.BulkShipmentProvider = (*Client)(nil)
	_ courier.BalanceProvider      = (*Client)(nil)
)

func New(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("STEADFAST_API_KEY")
	}
	if cfg.SecretKey == "" {
		cfg.SecretKey = os.Getenv("STEADFAST_SECRET_KEY")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = os.Getenv("STEADFAST_BASE_URL")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, &courier.Error{Kind: courier.ErrorKindConfiguration, Message: "steadfast api key is required"}
	}
	if strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, &courier.Error{Kind: courier.ErrorKindConfiguration, Message: "steadfast secret key is required"}
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = DefaultURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.RetryWait == 0 {
		cfg.RetryWait = 300 * time.Millisecond
	}
	if cfg.APIKeyHeader == "" {
		cfg.APIKeyHeader = "Api-Key"
	}
	if cfg.SecretKeyHeader == "" {
		cfg.SecretKeyHeader = "Secret-Key"
	}

	return &Client{
		apiKey:          strings.TrimSpace(cfg.APIKey),
		secretKey:       strings.TrimSpace(cfg.SecretKey),
		baseURL:         strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		httpClient:      cfg.HTTPClient,
		maxRetries:      cfg.MaxRetries,
		retryWait:       cfg.RetryWait,
		apiKeyHeader:    cfg.APIKeyHeader,
		secretKeyHeader: cfg.SecretKeyHeader,
	}, nil
}

func (c *Client) CreateShipment(ctx context.Context, req courier.CreateShipmentRequest) (*courier.Shipment, error) {
	payload, err := buildOrderRequest(req)
	if err != nil {
		return nil, err
	}

	var out orderEnvelopeResponse
	raw, err := c.do(ctx, http.MethodPost, "/create_order", nil, payload, &out)
	if err != nil {
		return nil, err
	}
	return mapOrderResponse(out.Consignment, raw), nil
}

func (c *Client) CreateShipments(ctx context.Context, reqs []courier.CreateShipmentRequest) (*courier.BulkShipmentResult, error) {
	if len(reqs) == 0 {
		return nil, &courier.Error{Kind: courier.ErrorKindValidation, Field: "shipments", Message: "shipments cannot be empty"}
	}
	if len(reqs) > 500 {
		return nil, &courier.Error{Kind: courier.ErrorKindValidation, Field: "shipments", Message: "steadfast accepts at most 500 shipments per bulk request"}
	}

	orders := make([]orderRequest, 0, len(reqs))
	for i, req := range reqs {
		payload, err := buildOrderRequest(req)
		if err != nil {
			return nil, &courier.Error{Kind: courier.ErrorKindValidation, Field: "shipments", Message: fmt.Sprintf("shipment %d: %v", i+1, err), Err: err}
		}
		orders = append(orders, payload)
	}

	encodedOrders, err := json.Marshal(orders)
	if err != nil {
		return nil, &courier.Error{Kind: courier.ErrorKindValidation, Message: "encode bulk order data", Err: err}
	}

	var out []bulkOrderItemResponse
	raw, err := c.do(ctx, http.MethodPost, "/create_order/bulk-order", nil, bulkOrderRequest{Data: string(encodedOrders)}, &out)
	if err != nil {
		return nil, err
	}

	results := make([]courier.BulkShipmentItemResult, 0, len(out))
	for _, item := range out {
		itemRaw, _ := json.Marshal(item)
		success := strings.EqualFold(item.Status, "success") || item.Error == ""
		var shipment *courier.Shipment
		if success {
			shipment = &courier.Shipment{
				Provider:        ProviderName,
				ConsignmentID:   stringify(item.ConsignmentID),
				MerchantOrderID: item.Invoice,
				TrackingCode:    item.TrackingCode,
				Status:          item.Status,
				Recipient: courier.Recipient{
					Name:    item.RecipientName,
					Phone:   item.RecipientPhone,
					Address: item.RecipientAddress,
				},
				CODAmount: moneyFromProviderAmount(item.CODAmount),
				Note:      item.Note,
				Raw:       itemRaw,
			}
		}
		results = append(results, courier.BulkShipmentItemResult{
			Shipment: shipment,
			Success:  success,
			Error:    item.Error,
			Raw:      itemRaw,
		})
	}

	return &courier.BulkShipmentResult{Results: results, Raw: raw}, nil
}

func (c *Client) TrackShipment(ctx context.Context, ref courier.TrackingReference) (*courier.TrackingStatus, error) {
	path, err := trackingPath(ref)
	if err != nil {
		return nil, err
	}

	var out statusResponse
	raw, err := c.do(ctx, http.MethodGet, path, nil, nil, &out)
	if err != nil {
		return nil, err
	}
	return &courier.TrackingStatus{
		Provider:       ProviderName,
		Reference:      ref,
		Status:         normalizeStatus(out.DeliveryStatus),
		ProviderStatus: out.DeliveryStatus,
		Raw:            raw,
	}, nil
}

func (c *Client) CurrentBalance(ctx context.Context) (*courier.Balance, error) {
	var out balanceResponse
	raw, err := c.do(ctx, http.MethodGet, "/get_balance", nil, nil, &out)
	if err != nil {
		return nil, err
	}
	return &courier.Balance{Provider: ProviderName, Amount: moneyFromProviderAmount(out.CurrentBalance), Raw: raw}, nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, payload any, out any) (json.RawMessage, error) {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, &courier.Error{Kind: courier.ErrorKindValidation, Message: "encode request payload", Err: err}
		}
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(c.retryWait * time.Duration(1<<(attempt-1)))
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, &courier.Error{Kind: courier.ErrorKindNetwork, Message: ctx.Err().Error(), Err: ctx.Err()}
			case <-timer.C:
			}
		}

		raw, err := c.doOnce(ctx, method, path, query, body, out)
		if err == nil {
			return raw, nil
		}
		lastErr = err
		if !retryable(err) {
			break
		}
	}
	return nil, lastErr
}

func (c *Client) doOnce(ctx context.Context, method, path string, query url.Values, body []byte, out any) (json.RawMessage, error) {
	endpoint := c.baseURL + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, &courier.Error{Kind: courier.ErrorKindConfiguration, Message: "build request", Err: err}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set(c.apiKeyHeader, c.apiKey)
	req.Header.Set(c.secretKeyHeader, c.secretKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &courier.Error{Kind: courier.ErrorKindNetwork, Message: err.Error(), Err: err}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &courier.Error{Kind: courier.ErrorKindNetwork, Message: "read response body", Err: err}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, responseError(resp.StatusCode, raw)
	}
	if out == nil {
		return raw, nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return nil, &courier.Error{Kind: courier.ErrorKindAPI, Message: "invalid json response", Err: err}
	}
	return raw, nil
}

func responseError(status int, raw []byte) error {
	message := strings.TrimSpace(string(raw))
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err == nil {
		if v, ok := body["message"].(string); ok && v != "" {
			message = v
		} else if v, ok := body["error"].(string); ok && v != "" {
			message = v
		}
	}
	if message == "" {
		message = http.StatusText(status)
	}
	kind := courier.ErrorKindAPI
	switch status {
	case http.StatusUnauthorized:
		kind = courier.ErrorKindAuthentication
	case http.StatusNotFound:
		kind = courier.ErrorKindNotFound
	}
	return &courier.Error{Kind: kind, StatusCode: status, Message: message}
}

func retryable(err error) bool {
	var cerr *courier.Error
	if !errors.As(err, &cerr) {
		return false
	}
	if cerr.Kind == courier.ErrorKindNetwork {
		return true
	}
	return cerr.Kind == courier.ErrorKindAPI && (cerr.StatusCode == http.StatusTooManyRequests || cerr.StatusCode >= 500)
}
