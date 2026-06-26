# Steadfast Courier Python Client API Report

This report summarizes the public surface exposed by the Python implementation in `steadfast-python/steadfast`, then translates it into suggestions for a generic Go courier abstraction that can support Steadfast and other courier providers.

## Current Steadfast Wire Behavior

The current Go implementation follows this Steadfast wire behavior:

- `placeOrder($data)` posts to `/create_order`.
- `bulkCreateOrders($data)` posts to `/create_order/bulk-order` with payload `{ "data": json_encode($data) }`.
- `checkDeliveryStatusByConsignmentId($id)` gets `/status_by_cid/{id}`.
- `checkDeliveryStatusByInvoiceId($id)` gets `/status_by_invoice/{id}`.
- `checkDeliveryStatusByTrackingCode($id)` gets `/status_by_trackingcode/{id}`.
- `getCurrentBalance()` gets `/get_balance`.

Authoritative headers:

- `Api-Key`
- `Secret-Key`
- `Content-Type: application/json`

Important differences from the Python implementation:

- Single-order response is wrapped as `{ "status": 200, "message": "...", "consignment": {...} }`.
- Bulk creation endpoint is `/create_order/bulk-order`, not `/create_bulk_order`.
- Bulk creation response is a top-level array, not `{ "results": [...] }`.
- Default base URL is `https://portal.packzy.com/api/v1`; keep base URL configurable.
- Payment, return, and location APIs should remain optional generic capabilities.

## High-Level Shape

The SDK exposes a main `SteadfastClient` facade. The facade lazily creates six module clients that all share one internal HTTP client:

- `client.orders`
- `client.tracking`
- `client.balance`
- `client.returns`
- `client.payments`
- `client.locations`

The package root also exports response models and exception types, but the user-facing operational methods live inside the module properties above.

## Client Construction

```python
SteadfastClient(api_key=None, secret_key=None, base_url=None)
```

Behavior:

- Reads credentials from explicit parameters first.
- Falls back to `STEADFAST_API_KEY` and `STEADFAST_SECRET_KEY`.
- Defaults `base_url` to `https://api.steadfast.io/v1`.
- Raises `ConfigurationError` if either credential is missing.
- Lazily initializes module clients on first property access.

Important observation: the Python implementation validates that `api_key` and `secret_key` exist, but the internal `HTTPClient` is only initialized with `base_url`. The credentials are not attached to requests in the code reviewed. A Go implementation should deliberately wire authentication headers according to the real Steadfast API contract.

## Exposed Modules And Methods

### Orders

Module property:

```python
client.orders
```

Methods:

```python
create(
    invoice: str,
    recipient_name: str,
    recipient_phone: str,
    recipient_address: str,
    cod_amount: int | float,
    delivery_type: int = 0,
    alternative_phone: str | None = None,
    recipient_email: str | None = None,
    note: str | None = None,
    item_description: str | None = None,
    total_lot: int | None = None,
) -> Order
```

Creates a single order by posting to `/create_order`.

Required fields:

- `invoice`
- `recipient_name`
- `recipient_phone`
- `recipient_address`
- `cod_amount`

Optional fields:

- `delivery_type`, defaults to `0`
- `alternative_phone`
- `recipient_email`
- `note`
- `item_description`
- `total_lot`

Returns `Order`:

```text
consignment_id, invoice, tracking_code, recipient_name, recipient_phone,
recipient_address, cod_amount, status, note, created_at, updated_at
```

```python
create_bulk(orders: list[dict]) -> BulkOrderResponse
```

Creates up to 500 orders by posting to `/create_bulk_order`.

Behavior:

- Rejects an empty list.
- Rejects more than 500 orders.
- Validates each order using the same single-order rules.
- Adds the failing item index to validation errors.

Returns `BulkOrderResponse` containing a list of `BulkOrderResult`:

```text
invoice, recipient_name, recipient_address, recipient_phone, cod_amount,
note, consignment_id, tracking_code, status, error
```

Private helper:

```python
_validate_order(**kwargs) -> dict
```

This is not part of the public client contract, but it documents the expected order payload shape.

### Tracking

Module property:

```python
client.tracking
```

Methods:

```python
get_status_by_consignment_id(consignment_id: int) -> OrderStatus
```

Fetches status from `/status_by_cid/{consignment_id}`.

```python
get_status_by_invoice(invoice: str) -> OrderStatus
```

Fetches status from `/status_by_invoice/{invoice}`.

```python
get_status_by_tracking_code(tracking_code: str) -> OrderStatus
```

Fetches status from `/status_by_trackingcode/{tracking_code}`.

All three return `OrderStatus`:

```text
status, delivery_status
```

### Balance

Module property:

```python
client.balance
```

Method:

```python
get_current_balance() -> Balance
```

Fetches account balance from `/get_balance`.

Returns `Balance`:

```text
status, current_balance
```

### Returns

Module property:

```python
client.returns
```

Methods:

```python
create(
    identifier: int | str,
    identifier_type: str = "consignment_id",
    reason: str = "",
) -> ReturnRequest
```

Creates a return request by posting to `/return-request/store`.

Supported `identifier_type` values:

- `consignment_id`
- `invoice`
- `tracking_code`

Validation differs by type:

- `consignment_id` must be a positive integer.
- `invoice` must match the invoice format.
- `tracking_code` is accepted after identifier type validation; the module does not add additional tracking-code validation here.

```python
get(return_request_id: int) -> ReturnRequest
```

Fetches one return request from `/return-request/{return_request_id}`.

```python
list() -> ReturnRequestList
```

Fetches return requests from `/return-request/list`.

Returns `ReturnRequest` or `ReturnRequestList`:

```text
id, user_id, consignment_id, reason, status, created_at, updated_at
```

### Payments

Module property:

```python
client.payments
```

Methods:

```python
list() -> PaymentList
```

Fetches payments from `/payment/list`.

Returns `PaymentList`:

```text
data: []Payment
```

Each `Payment` has:

```text
id, amount, created_at, updated_at
```

```python
get(payment_id: int) -> PaymentDetails
```

Fetches payment details from `/payment/{payment_id}`.

Returns `PaymentDetails`:

```text
id, amount, consignments, created_at, updated_at
```

### Locations

Module property:

```python
client.locations
```

Method:

```python
get_police_stations() -> PoliceStationList
```

Fetches police station data from `/location/police-stations`.

Returns `PoliceStationList`:

```text
data: []PoliceStation
```

Each `PoliceStation` has:

```text
id, name, location
```

## Validation Rules In The Python SDK

The implementation validates inputs before calling the API:

- `invoice`: non-empty string, only letters, numbers, hyphens, and underscores.
- `phone`: non-empty string, normalized to digits only, must be exactly 11 digits.
- `recipient_name`: non-empty string, trimmed, maximum 100 characters.
- `recipient_address`: non-empty string, trimmed, maximum 250 characters.
- `recipient_email`: basic email format validation.
- `cod_amount`: numeric and greater than or equal to zero.
- `delivery_type`: `0` for home delivery or `1` for point delivery.
- `consignment_id`: positive integer.
- `identifier_type`: one of `consignment_id`, `invoice`, or `tracking_code`.

For a Go port, keep validation close to request DTOs, but avoid baking Steadfast-specific validation into the generic interface. Other couriers may have different phone, address, service-type, parcel-weight, area-code, or merchant-order-id requirements.

## Error Model

The Python SDK defines these error categories:

- `ConfigurationError`: missing client configuration.
- `ValidationError`: local input validation failed, optionally tied to a field.
- `AuthenticationError`: HTTP 401.
- `NotFoundError`: HTTP 404.
- `APIError`: non-401/404 API failure or invalid JSON response.
- `NetworkError`: connection, timeout, or request-level failure.

The internal HTTP client:

- Has a default timeout of 30 seconds.
- Retries connection errors and timeouts.
- Uses `max_retries=3`.
- Uses exponential backoff starting at `0.3` seconds.
- Sets `Content-Type: application/json` for requests with a JSON body.

## Suggested Go Interface Design

Keep the generic interface focused on business capabilities instead of mirroring every provider endpoint. A useful starting point:

```go
type CourierClient interface {
    CreateShipment(ctx context.Context, req CreateShipmentRequest) (*Shipment, error)
    TrackShipment(ctx context.Context, ref TrackingReference) (*TrackingStatus, error)
}
```

Then expose optional capability interfaces for provider-specific or less common features:

```go
type BulkShipmentProvider interface {
    CreateShipments(ctx context.Context, reqs []CreateShipmentRequest) (*BulkShipmentResult, error)
}

type BalanceProvider interface {
    CurrentBalance(ctx context.Context) (*Balance, error)
}

type PaymentProvider interface {
    ListPayments(ctx context.Context, opts ListOptions) (*PaymentPage, error)
    GetPayment(ctx context.Context, paymentID string) (*PaymentDetails, error)
}

type LocationProvider interface {
    ListServiceAreas(ctx context.Context, opts ListOptions) (*ServiceAreaPage, error)
}

type ReturnProvider interface {
    GetReturn(ctx context.Context, returnID string) (*ReturnRequest, error)
    ListReturns(ctx context.Context, opts ListOptions) (*ReturnPage, error)
}
```

This keeps your core abstraction portable. Steadfast supports balance, payment listing, payment details, police stations, and return listing, but another courier may not.

## Suggested Go DTO Mapping

Steadfast-specific adapter methods can map generic DTOs to the provider payload:

```go
type CreateShipmentRequest struct {
    MerchantOrderID string
    Recipient       Recipient
    CODAmount       decimal.Decimal
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
    Provider          string
    ConsignmentID     string
    MerchantOrderID   string
    TrackingCode      string
    Status            string
    Raw               json.RawMessage
}

type TrackingReference struct {
    Type  TrackingReferenceType
    Value string
}
```

For Steadfast:

- `MerchantOrderID` maps to `invoice`.
- `Recipient.Name` maps to `recipient_name`.
- `Recipient.Phone` maps to `recipient_phone`.
- `Recipient.Address` maps to `recipient_address`.
- `CODAmount` maps to `cod_amount`.
- `DeliveryType` maps to Steadfast's `0` or `1`.
- `Shipment.ConsignmentID` maps from `consignment_id`.
- `Shipment.TrackingCode` maps from `tracking_code`.

## Go Implementation Tips

- Use `context.Context` in every public method.
- Use one `http.Client` per provider client; inject it for tests.
- Keep credentials in a provider-specific config struct, for example `SteadfastConfig`.
- Implement retries only for safe failures such as network errors, timeouts, and possibly 429 or 5xx responses. Be careful retrying order creation unless the provider supports idempotency by invoice or idempotency keys.
- Store raw provider responses on generic response structs. This is extremely useful when a courier exposes fields your abstraction does not yet understand.
- Use typed enums for generic concepts, but keep a provider escape hatch through `ProviderFields`.
- Normalize statuses into your own status enum, but preserve the provider status string.
- Treat bulk creation as a partial-success operation. A batch can return mixed success and failure results.
- Make pagination explicit for list operations, even though the current Python module does not expose pagination parameters.
- Avoid naming generic methods after Steadfast concepts such as `Invoice` or `PoliceStation`. Prefer names like `MerchantOrderID`, `ServiceArea`, or `Location`.
- Use money-safe representation for COD and payments. In Go, prefer integer minor units or a decimal library over `float64`.
- Add table-driven tests for request mapping, validation, error classification, and bulk partial failures.

## Possible Package Layout In Go

```text
courier/
  client.go              # Generic interfaces and shared DTOs
  errors.go              # Generic error categories
  status.go              # Status normalization
  steadfast/
    client.go            # Steadfast implementation
    models.go            # Steadfast request/response payloads
    mapper.go            # Generic <-> Steadfast mapping
    validation.go        # Steadfast-specific validation
    errors.go            # Steadfast error mapping
```

This lets application code depend on `courier.CourierClient`, while provider-specific code lives under `courier/steadfast`.

## Key Caveats From The Python Implementation

- Authentication appears incomplete in the reviewed code because credentials are not passed into request headers.
- The SDK uses `float` for money-like values, which is not ideal for Go.
- Tracking has three separate methods, but a generic interface should use one `TrackShipment(ctx, TrackingReference)` method.
- Return creation accepts different identifier types. Model this explicitly instead of overloading raw strings.
- Location support is Steadfast-specific and should probably be an optional capability.
- Payment and balance support are account/merchant features, not shipment features, so keep them outside the core courier interface.
