# courier-go

`courier-go` is a Go courier abstraction with a Steadfast implementation and room for other courier providers.

Module path:

```bash
go get github.com/nokibsarkar/courier-go
```

## Features

- Generic shipment interface for courier providers.
- Steadfast implementation for:
  - creating a shipment
  - tracking by consignment ID, merchant order ID, or tracking code
  - creating bulk shipments, exposed as an optional capability
  - reading current balance
- Environment-variable configuration for simple apps.
- Explicit config and injected `*http.Client` for SaaS or multi-tenant apps.
- Provider-neutral error categories.
- Raw provider responses preserved on returned DTOs.

## Steadfast Configuration

The Steadfast client reads these environment variables when explicit config fields are empty:

```bash
export STEADFAST_API_KEY="your-api-key"
export STEADFAST_SECRET_KEY="your-secret-key"
export STEADFAST_BASE_URL="https://portal.packzy.com/api/v1"
```

`STEADFAST_BASE_URL` is optional. The default is:

```text
https://portal.packzy.com/api/v1
```

The Steadfast client sends these headers:

```text
Api-Key: <api key>
Secret-Key: <secret key>
Content-Type: application/json
```

## Basic Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	courier "github.com/nokibsarkar/courier-go"
	"github.com/nokibsarkar/courier-go/steadfast"
	"github.com/shopspring/decimal"
)

func main() {
	client, err := steadfast.New(steadfast.Config{})
	if err != nil {
		log.Fatal(err)
	}

	shipment, err := client.CreateShipment(context.Background(), courier.CreateShipmentRequest{
		MerchantOrderID: "ORD-1001",
		Recipient: courier.Recipient{
			Name:    "John Doe",
			Phone:   "01711111111",
			Address: "Dhanmondi, Dhaka",
		},
		CODAmount: courier.Money{Value: decimal.NewFromInt(1000), Currency: "BDT"},
		Note:      "Handle with care",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(shipment.ConsignmentID, shipment.TrackingCode)
}
```

## SaaS / Multi-Tenant Usage

For SaaS apps, avoid global configuration. Create one client per tenant or merchant account and inject your own `*http.Client`.

```go
package courierfactory

import (
	"net/http"
	"time"

	"github.com/nokibsarkar/courier-go/steadfast"
)

type TenantCourierConfig struct {
	APIKey    string
	SecretKey string
	BaseURL   string
}

func NewSteadfastTenantClient(cfg TenantCourierConfig) (*steadfast.Client, error) {
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
```

Explicit config always wins over environment variables, so this is safe for tenant-specific credentials.

## Generic Interface

Application services can depend on the provider-neutral interface:

```go
type ShippingService struct {
	courier courier.CourierClient
}

func NewShippingService(client courier.CourierClient) *ShippingService {
	return &ShippingService{courier: client}
}
```

Optional provider capabilities can be checked at runtime:

```go
if balanceProvider, ok := courier.AsBalanceProvider(client); ok {
	balance, err := balanceProvider.CurrentBalance(ctx)
	_ = balance
	_ = err
}

if bulkProvider, ok := courier.AsBulkShipmentProvider(client); ok {
	result, err := bulkProvider.CreateShipments(ctx, []courier.CreateShipmentRequest{})
	_ = result
	_ = err
}

if paymentProvider, ok := courier.AsPaymentProvider(client); ok {
	payments, err := paymentProvider.ListPayments(ctx, courier.ListOptions{})
	_ = payments
	_ = err
}

if locationProvider, ok := courier.AsLocationProvider(client); ok {
	areas, err := locationProvider.ListServiceAreas(ctx, courier.ListOptions{})
	_ = areas
	_ = err
}

if returnProvider, ok := courier.AsReturnProvider(client); ok {
	returns, err := returnProvider.ListReturns(ctx, courier.ListOptions{})
	_ = returns
	_ = err
}
```

Not every courier supports every optional interface. The core `CourierClient` remains focused on single-shipment creation and tracking.

## Bulk Shipments

```go
bulkProvider, ok := courier.AsBulkShipmentProvider(client)
if !ok {
	// Fall back to creating shipments one by one.
}

result, err := bulkProvider.CreateShipments(ctx, []courier.CreateShipmentRequest{
	{
		MerchantOrderID: "ORD-1001",
		Recipient: courier.Recipient{
			Name:    "John Doe",
			Phone:   "01711111111",
			Address: "Dhaka",
		},
		CODAmount: courier.Money{Value: decimal.NewFromInt(1000), Currency: "BDT"},
	},
	{
		MerchantOrderID: "ORD-1002",
		Recipient: courier.Recipient{
			Name:    "Jane Doe",
			Phone:   "01811111111",
			Address: "Chattogram",
		},
		CODAmount: courier.Money{Value: decimal.NewFromInt(1500), Currency: "BDT"},
	},
})
```

Steadfast bulk creation uses:

- endpoint: `/create_order/bulk-order`
- body: `{ "data": "<JSON encoded order array>" }`
- response: top-level array of per-order results

## Tracking

```go
status, err := client.TrackShipment(ctx, courier.TrackingReference{
	Type:  courier.TrackingReferenceMerchantOrder,
	Value: "ORD-1001",
})
```

Supported reference types:

- `courier.TrackingReferenceConsignmentID`
- `courier.TrackingReferenceMerchantOrder`
- `courier.TrackingReferenceTrackingCode`

## Balance

```go
balance, err := client.CurrentBalance(ctx)
```

`Money.Value` stores the amount as `decimal.Decimal`. Courier implementations convert this value to their provider-specific request format and convert provider responses back into `Money`.

## Errors

Errors returned by the package are usually `*courier.Error`:

```go
shipment, err := client.CreateShipment(ctx, req)
if err != nil {
	if courier.IsKind(err, courier.ErrorKindValidation) {
		// Fix the request payload.
	}
	return err
}
```

Error kinds:

- `configuration`
- `validation`
- `authentication`
- `not_found`
- `api`
- `network`

## Optional Interfaces

The package defines optional interfaces for bulk shipment creation, account balance, payments, locations, and returns. Provider implementations can opt into any of these without changing the core `CourierClient` contract.

## License

This project is licensed under the GNU General Public License v3.0. See [LICENSE.md](LICENSE.md).
