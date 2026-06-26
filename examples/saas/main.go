package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	courier "github.com/nokibsarkar/courier-go"
	"github.com/nokibsarkar/courier-go/steadfast"
)

type TenantCourierConfig struct {
	APIKey    string
	SecretKey string
	BaseURL   string
}

func newTenantClient(cfg TenantCourierConfig) (*steadfast.Client, error) {
	return steadfast.New(steadfast.Config{
		APIKey:    cfg.APIKey,
		SecretKey: cfg.SecretKey,
		BaseURL:   cfg.BaseURL,
		HTTPClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		MaxRetries: 1,
		RetryWait:  250 * time.Millisecond,
	})
}

func main() {
	client, err := newTenantClient(TenantCourierConfig{
		APIKey:    "tenant-api-key",
		SecretKey: "tenant-secret-key",
		BaseURL:   steadfast.DefaultURL,
	})
	if err != nil {
		log.Fatal(err)
	}

	status, err := client.TrackShipment(context.Background(), courier.TrackingReference{
		Type:  courier.TrackingReferenceMerchantOrder,
		Value: "ORD-1001",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(status.ProviderStatus)
}
