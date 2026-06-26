package courier

import (
	"context"
	"testing"
)

type optionalProvider struct{}

func (optionalProvider) ListPayments(context.Context, ListOptions) (*PaymentPage, error) {
	return nil, nil
}

func (optionalProvider) CreateShipments(context.Context, []CreateShipmentRequest) (*BulkShipmentResult, error) {
	return nil, nil
}

func (optionalProvider) GetPayment(context.Context, string) (*PaymentDetails, error) {
	return nil, nil
}

func (optionalProvider) ListServiceAreas(context.Context, ListOptions) (*ServiceAreaPage, error) {
	return nil, nil
}

func (optionalProvider) CreateReturn(context.Context, CreateReturnRequest) (*ReturnRequest, error) {
	return nil, nil
}

func (optionalProvider) GetReturn(context.Context, string) (*ReturnRequest, error) {
	return nil, nil
}

func (optionalProvider) ListReturns(context.Context, ListOptions) (*ReturnPage, error) {
	return nil, nil
}

func TestOptionalCapabilityHelpers(t *testing.T) {
	provider := optionalProvider{}

	if _, ok := AsPaymentProvider(provider); !ok {
		t.Fatal("expected payment provider")
	}
	if _, ok := AsBulkShipmentProvider(provider); !ok {
		t.Fatal("expected bulk shipment provider")
	}
	if _, ok := AsLocationProvider(provider); !ok {
		t.Fatal("expected location provider")
	}
	if _, ok := AsReturnProvider(provider); !ok {
		t.Fatal("expected return provider")
	}
	if _, ok := AsBalanceProvider(provider); ok {
		t.Fatal("did not expect balance provider")
	}
}
