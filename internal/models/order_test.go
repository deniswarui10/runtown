package models

import (
	"testing"
	"time"
)

func TestOrder_Validate(t *testing.T) {
	tests := []struct {
		name    string
		order   Order
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid order",
			order: Order{
				OrderNumber:  "ORD-20240101-123456",
				TotalAmount:  2500,
				Status:       OrderCompleted,
				BillingEmail: "test@example.com",
				BillingName:  "John Doe",
			},
			wantErr: false,
		},
		{
			name: "invalid order number - empty",
			order: Order{
				OrderNumber:  "",
				TotalAmount:  2500,
				Status:       OrderCompleted,
				BillingEmail: "test@example.com",
				BillingName:  "John Doe",
			},
			wantErr: true,
			errMsg:  "order number is required",
		},
		{
			name: "invalid order number - format",
			order: Order{
				OrderNumber:  "INVALID-123",
				TotalAmount:  2500,
				Status:       OrderCompleted,
				BillingEmail: "test@example.com",
				BillingName:  "John Doe",
			},
			wantErr: true,
			errMsg:  "order number format is invalid",
		},
		{
			name: "invalid total amount - negative",
			order: Order{
				OrderNumber:  "ORD-20240101-123456",
				TotalAmount:  -100,
				Status:       OrderCompleted,
				BillingEmail: "test@example.com",
				BillingName:  "John Doe",
			},
			wantErr: true,
			errMsg:  "total amount cannot be negative",
		},
		{
			name: "invalid status",
			order: Order{
				OrderNumber:  "ORD-20240101-123456",
				TotalAmount:  2500,
				Status:       "invalid",
				BillingEmail: "test@example.com",
				BillingName:  "John Doe",
			},
			wantErr: true,
			errMsg:  "invalid order status",
		},
		{
			name: "invalid billing email - empty",
			order: Order{
				OrderNumber:  "ORD-20240101-123456",
				TotalAmount:  2500,
				Status:       OrderCompleted,
				BillingEmail: "",
				BillingName:  "John Doe",
			},
			wantErr: true,
			errMsg:  "billing email is required",
		},
		{
			name: "invalid billing name - empty",
			order: Order{
				OrderNumber:  "ORD-20240101-123456",
				TotalAmount:  2500,
				Status:       OrderCompleted,
				BillingEmail: "test@example.com",
				BillingName:  "",
			},
			wantErr: true,
			errMsg:  "billing name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.order.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Order.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Order.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestOrder_StatusChecks(t *testing.T) {
	tests := []struct {
		name   string
		status OrderStatus
		checks map[string]bool
	}{
		{
			name:   "pending order",
			status: OrderPending,
			checks: map[string]bool{
				"IsPending":      true,
				"IsCompleted":    false,
				"IsCancelled":    false,
				"IsRefunded":     false,
				"CanBeCancelled": true,
				"CanBeRefunded":  false,
				"CanBeCompleted": true,
			},
		},
		{
			name:   "completed order",
			status: OrderCompleted,
			checks: map[string]bool{
				"IsPending":      false,
				"IsCompleted":    true,
				"IsCancelled":    false,
				"IsRefunded":     false,
				"CanBeCancelled": false,
				"CanBeRefunded":  true,
				"CanBeCompleted": false,
			},
		},
		{
			name:   "cancelled order",
			status: OrderCancelled,
			checks: map[string]bool{
				"IsPending":      false,
				"IsCompleted":    false,
				"IsCancelled":    true,
				"IsRefunded":     false,
				"CanBeCancelled": false,
				"CanBeRefunded":  false,
				"CanBeCompleted": false,
			},
		},
		{
			name:   "refunded order",
			status: OrderRefunded,
			checks: map[string]bool{
				"IsPending":      false,
				"IsCompleted":    false,
				"IsCancelled":    false,
				"IsRefunded":     true,
				"CanBeCancelled": false,
				"CanBeRefunded":  false,
				"CanBeCompleted": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := Order{Status: tt.status}
			
			if got := order.IsPending(); got != tt.checks["IsPending"] {
				t.Errorf("Order.IsPending() = %v, want %v", got, tt.checks["IsPending"])
			}
			if got := order.IsCompleted(); got != tt.checks["IsCompleted"] {
				t.Errorf("Order.IsCompleted() = %v, want %v", got, tt.checks["IsCompleted"])
			}
			if got := order.IsCancelled(); got != tt.checks["IsCancelled"] {
				t.Errorf("Order.IsCancelled() = %v, want %v", got, tt.checks["IsCancelled"])
			}
			if got := order.IsRefunded(); got != tt.checks["IsRefunded"] {
				t.Errorf("Order.IsRefunded() = %v, want %v", got, tt.checks["IsRefunded"])
			}
			if got := order.CanBeCancelled(); got != tt.checks["CanBeCancelled"] {
				t.Errorf("Order.CanBeCancelled() = %v, want %v", got, tt.checks["CanBeCancelled"])
			}
			if got := order.CanBeRefunded(); got != tt.checks["CanBeRefunded"] {
				t.Errorf("Order.CanBeRefunded() = %v, want %v", got, tt.checks["CanBeRefunded"])
			}
			if got := order.CanBeCompleted(); got != tt.checks["CanBeCompleted"] {
				t.Errorf("Order.CanBeCompleted() = %v, want %v", got, tt.checks["CanBeCompleted"])
			}
		})
	}
}

func TestOrder_TotalAmountInDollars(t *testing.T) {
	order := Order{TotalAmount: 2550} // $25.50
	
	expected := 25.50
	if got := order.TotalAmountInDollars(); got != expected {
		t.Errorf("Order.TotalAmountInDollars() = %v, want %v", got, expected)
	}
}

func TestOrder_IsExpired(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name               string
		order              Order
		expirationDuration time.Duration
		want               bool
	}{
		{
			name: "pending order not expired",
			order: Order{
				Status:    OrderPending,
				CreatedAt: now.Add(-10 * time.Minute),
			},
			expirationDuration: 15 * time.Minute,
			want:               false,
		},
		{
			name: "pending order expired",
			order: Order{
				Status:    OrderPending,
				CreatedAt: now.Add(-20 * time.Minute),
			},
			expirationDuration: 15 * time.Minute,
			want:               true,
		},
		{
			name: "completed order never expires",
			order: Order{
				Status:    OrderCompleted,
				CreatedAt: now.Add(-20 * time.Minute),
			},
			expirationDuration: 15 * time.Minute,
			want:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.order.IsExpired(tt.expirationDuration); got != tt.want {
				t.Errorf("Order.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateOrderNumber(t *testing.T) {
	orderNumber := GenerateOrderNumber()
	
	// Check format: ORD-YYYYMMDD-XXXXXX
	if !orderNumberRegex.MatchString(orderNumber) {
		t.Errorf("GenerateOrderNumber() = %v, does not match expected format", orderNumber)
	}
	
	// Generate another one to ensure they're different
	orderNumber2 := GenerateOrderNumber()
	if orderNumber == orderNumber2 {
		t.Errorf("GenerateOrderNumber() generated duplicate order numbers")
	}
}

func TestOrder_GetStatusDisplayName(t *testing.T) {
	tests := []struct {
		name   string
		status OrderStatus
		want   string
	}{
		{
			name:   "pending status",
			status: OrderPending,
			want:   "Pending Payment",
		},
		{
			name:   "completed status",
			status: OrderCompleted,
			want:   "Completed",
		},
		{
			name:   "cancelled status",
			status: OrderCancelled,
			want:   "Cancelled",
		},
		{
			name:   "refunded status",
			status: OrderRefunded,
			want:   "Refunded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := Order{Status: tt.status}
			if got := order.GetStatusDisplayName(); got != tt.want {
				t.Errorf("Order.GetStatusDisplayName() = %v, want %v", got, tt.want)
			}
		})
	}
}