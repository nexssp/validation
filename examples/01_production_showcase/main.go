package main

import (
	"context"
	"fmt"
	"time"

	"github.com/nexssp/kernel/action"
	"github.com/nexssp/kernel/xerr"
	"github.com/nexssp/validation"
)

type AddressDTO struct {
	Street  string `json:"street" validate:"required,min=3"`
	City    string `json:"city" validate:"required"`
	Country string `json:"country" validate:"required,len=2"` // ISO-2 country code
}

type OrderItemDTO struct {
	SKU      string `json:"sku" validate:"required,alphanum"`
	Quantity int    `json:"quantity" validate:"required,gt=0"`
	PriceUSD int64  `json:"price_usd_cents" validate:"required,gt=0"`
}

type CheckoutReq struct {
	CustomerEmail string         `json:"customer_email" validate:"required,email"`
	Shipping      AddressDTO     `json:"shipping_address" validate:"required"`
	Items         []OrderItemDTO `json:"items" validate:"required,min=1,dive"`
	Notes         string         `json:"notes,omitempty"`
}

// Custom semantic validation logic (executed after tag validation)
func (c *CheckoutReq) Validate() error {
	var totalCents int64
	for _, item := range c.Items {
		totalCents += item.PriceUSD * int64(item.Quantity)
	}
	if totalCents < 500 { // $5.00 minimum checkout
		return fmt.Errorf("minimum checkout order total is $5.00 (got $%.2f)", float64(totalCents)/100)
	}
	return nil
}

type CheckoutRes struct {
	OrderID     string    `json:"order_id"`
	TotalUSD    float64   `json:"total_usd"`
	ProcessedAt time.Time `json:"processed_at"`
}

func main() {
	ctx := context.Background()

	// 1. Define action with pointer DTO (*CheckoutReq) for in-place string trimming
	processCheckout := validation.AutoValidate(
		action.New("checkout.process", func(_ context.Context, req *CheckoutReq) (CheckoutRes, error) {
			var totalCents int64
			for _, item := range req.Items {
				totalCents += item.PriceUSD * int64(item.Quantity)
			}

			// Notice: All strings (Email, Street, City, Country, Item SKUs)
			// were automatically trimmed in-place across nested structs and slices!
			return CheckoutRes{
				OrderID:     "ord_998877",
				TotalUSD:    float64(totalCents) / 100,
				ProcessedAt: time.Now().UTC(),
			}, nil
		}),
	).Build()

	// -------------------------------------------------------------------------
	// CASE 1: Valid Checkout with messy whitespace across nested fields
	// -------------------------------------------------------------------------
	fmt.Println("=================================================================")
	fmt.Println("🚀 TEST 1: Valid Request with Messy Whitespace across Nested Fields")
	fmt.Println("=================================================================")

	validReq := &CheckoutReq{
		CustomerEmail: "   alice@nexss.com   \t",
		Shipping: AddressDTO{
			Street:  "  100 Innovation Way, Suite 400  ",
			City:    "  San Francisco  ",
			Country: "  US  ",
		},
		Items: []OrderItemDTO{
			{SKU: "  PROD1001  ", Quantity: 2, PriceUSD: 2500}, // $25.00 x 2 = $50.00
		},
		Notes: "  Please deliver before 5 PM.  ",
	}

	fmt.Printf("BEFORE TRIMMING:\n  Email: %q\n  City: %q\n  SKU: %q\n\n",
		validReq.CustomerEmail, validReq.Shipping.City, validReq.Items[0].SKU)

	res, err := processCheckout.Do(ctx, validReq)
	if err != nil {
		panic(err)
	}

	fmt.Printf("AFTER AUTOMATIC RECURSIVE TRIMMING & VALIDATION:\n  Email: %q\n  City: %q\n  SKU: %q\n",
		validReq.CustomerEmail, validReq.Shipping.City, validReq.Items[0].SKU)
	fmt.Printf("✅ Order Processed: OrderID=%s Total=$%.2f\n\n", res.OrderID, res.TotalUSD)

	// -------------------------------------------------------------------------
	// CASE 2: Invalid Checkout triggering Tag + Custom Semantic Validation
	// -------------------------------------------------------------------------
	fmt.Println("=================================================================")
	fmt.Println("❌ TEST 2: Invalid Request Triggering Tag + Custom Validation")
	fmt.Println("=================================================================")

	invalidReq := &CheckoutReq{
		CustomerEmail: "bad-email-format",
		Shipping: AddressDTO{
			Street:  "St",  // Fails min=3
			City:    "",    // Fails required
			Country: "USA", // Fails len=2
		},
		Items: []OrderItemDTO{
			{SKU: "INVALID SKU!", Quantity: 0, PriceUSD: 100}, // Fails alphanum, gt=0; total $1.00 fails < $5.00
		},
	}

	_, err = processCheckout.Do(ctx, invalidReq)
	if err != nil {
		appErr := xerr.From(err)
		fmt.Printf("AppError Kind: [%s] | Message: %s\n", appErr.Kind, appErr.Message)
		fmt.Println("Structured Field Violations:")
		for i, detail := range appErr.ValidationDetails {
			fmt.Printf("  %2d. Field: %-20s Violation: %-12s Message: %s\n",
				i+1, detail.Field, detail.Validation, detail.Value)
		}
	}

	fmt.Println("=================================================================")
	fmt.Println("❌ TEST 3: Valid Struct Tags, Fails Custom Semantic Validation")
	fmt.Println("=================================================================")

	smallOrderReq := &CheckoutReq{
		CustomerEmail: "bob@nexss.com",
		Shipping: AddressDTO{
			Street:  "123 Main Street",
			City:    "Boston",
			Country: "US",
		},
		Items: []OrderItemDTO{
			{SKU: "ITEM100", Quantity: 1, PriceUSD: 250}, // $2.50 total (< $5.00 minimum)
		},
	}

	_, err = processCheckout.Do(ctx, smallOrderReq)
	if err != nil {
		appErr := xerr.From(err)
		fmt.Printf("AppError Kind: [%s] | Message: %s\n", appErr.Kind, appErr.Message)
		if appErr.Cause != nil {
			fmt.Printf("Custom Rule Cause: %v\n", appErr.Cause)
		}
	}
}
