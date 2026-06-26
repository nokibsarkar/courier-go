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

	fmt.Printf("consignment=%s tracking=%s\n", shipment.ConsignmentID, shipment.TrackingCode)
}
