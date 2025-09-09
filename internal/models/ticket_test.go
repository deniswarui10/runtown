package models

import (
	"testing"
	"time"
)

func TestTicketType_Validate(t *testing.T) {
	futureStart := time.Now().Add(1 * time.Hour)
	futureEnd := futureStart.Add(2 * time.Hour)

	tests := []struct {
		name       string
		ticketType TicketType
		wantErr    bool
		errMsg     string
	}{
		{
			name: "valid ticket type",
			ticketType: TicketType{
				Name:      "General Admission",
				Price:     2500, // $25.00
				Quantity:  100,
				SaleStart: futureStart,
				SaleEnd:   futureEnd,
			},
			wantErr: false,
		},
		{
			name: "invalid name - empty",
			ticketType: TicketType{
				Name:      "",
				Price:     2500,
				Quantity:  100,
				SaleStart: futureStart,
				SaleEnd:   futureEnd,
			},
			wantErr: true,
			errMsg:  "ticket type name is required",
		},
		{
			name: "invalid price - negative",
			ticketType: TicketType{
				Name:      "General Admission",
				Price:     -100,
				Quantity:  100,
				SaleStart: futureStart,
				SaleEnd:   futureEnd,
			},
			wantErr: true,
			errMsg:  "ticket price cannot be negative",
		},
		{
			name: "invalid quantity - zero",
			ticketType: TicketType{
				Name:      "General Admission",
				Price:     2500,
				Quantity:  0,
				SaleStart: futureStart,
				SaleEnd:   futureEnd,
			},
			wantErr: true,
			errMsg:  "ticket quantity must be greater than 0",
		},
		{
			name: "invalid sale period - start after end",
			ticketType: TicketType{
				Name:      "General Admission",
				Price:     2500,
				Quantity:  100,
				SaleStart: futureEnd,
				SaleEnd:   futureStart,
			},
			wantErr: true,
			errMsg:  "sale start date must be before sale end date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ticketType.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TicketType.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("TicketType.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestTicketType_IsAvailable(t *testing.T) {
	now := time.Now()
	pastStart := now.Add(-2 * time.Hour)
	futureEnd := now.Add(2 * time.Hour)

	tests := []struct {
		name       string
		ticketType TicketType
		want       bool
	}{
		{
			name: "available tickets",
			ticketType: TicketType{
				Quantity:  100,
				Sold:      50,
				SaleStart: pastStart,
				SaleEnd:   futureEnd,
			},
			want: true,
		},
		{
			name: "sold out",
			ticketType: TicketType{
				Quantity:  100,
				Sold:      100,
				SaleStart: pastStart,
				SaleEnd:   futureEnd,
			},
			want: false,
		},
		{
			name: "sale not started",
			ticketType: TicketType{
				Quantity:  100,
				Sold:      0,
				SaleStart: now.Add(1 * time.Hour),
				SaleEnd:   futureEnd,
			},
			want: false,
		},
		{
			name: "sale ended",
			ticketType: TicketType{
				Quantity:  100,
				Sold:      0,
				SaleStart: pastStart,
				SaleEnd:   now.Add(-1 * time.Hour),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ticketType.IsAvailable(); got != tt.want {
				t.Errorf("TicketType.IsAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTicketType_Available(t *testing.T) {
	tests := []struct {
		name       string
		ticketType TicketType
		want       int
	}{
		{
			name: "some available",
			ticketType: TicketType{
				Quantity: 100,
				Sold:     30,
			},
			want: 70,
		},
		{
			name: "none available",
			ticketType: TicketType{
				Quantity: 100,
				Sold:     100,
			},
			want: 0,
		},
		{
			name: "oversold (should return 0)",
			ticketType: TicketType{
				Quantity: 100,
				Sold:     110,
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ticketType.Available(); got != tt.want {
				t.Errorf("TicketType.Available() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTicketType_PriceInDollars(t *testing.T) {
	ticketType := TicketType{Price: 2550} // $25.50
	
	expected := 25.50
	if got := ticketType.PriceInDollars(); got != expected {
		t.Errorf("TicketType.PriceInDollars() = %v, want %v", got, expected)
	}
}

func TestTicketType_CanUpdateQuantity(t *testing.T) {
	ticketType := TicketType{
		Quantity: 100,
		Sold:     30,
	}

	tests := []struct {
		name        string
		newQuantity int
		want        bool
	}{
		{
			name:        "increase quantity",
			newQuantity: 150,
			want:        true,
		},
		{
			name:        "decrease to valid amount",
			newQuantity: 50,
			want:        true,
		},
		{
			name:        "decrease to sold amount",
			newQuantity: 30,
			want:        true,
		},
		{
			name:        "decrease below sold amount",
			newQuantity: 20,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ticketType.CanUpdateQuantity(tt.newQuantity); got != tt.want {
				t.Errorf("TicketType.CanUpdateQuantity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTicket_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ticket  Ticket
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid ticket",
			ticket: Ticket{
				QRCode: "QR123456789",
				Status: TicketActive,
			},
			wantErr: false,
		},
		{
			name: "invalid QR code - empty",
			ticket: Ticket{
				QRCode: "",
				Status: TicketActive,
			},
			wantErr: true,
			errMsg:  "QR code is required",
		},
		{
			name: "invalid status",
			ticket: Ticket{
				QRCode: "QR123456789",
				Status: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid ticket status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ticket.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Ticket.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Ticket.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestTicket_StatusChecks(t *testing.T) {
	tests := []struct {
		name   string
		status TicketStatus
		checks map[string]bool
	}{
		{
			name:   "active ticket",
			status: TicketActive,
			checks: map[string]bool{
				"IsActive":      true,
				"IsUsed":        false,
				"IsRefunded":    false,
				"CanBeUsed":     true,
				"CanBeRefunded": true,
			},
		},
		{
			name:   "used ticket",
			status: TicketUsed,
			checks: map[string]bool{
				"IsActive":      false,
				"IsUsed":        true,
				"IsRefunded":    false,
				"CanBeUsed":     false,
				"CanBeRefunded": false,
			},
		},
		{
			name:   "refunded ticket",
			status: TicketRefunded,
			checks: map[string]bool{
				"IsActive":      false,
				"IsUsed":        false,
				"IsRefunded":    true,
				"CanBeUsed":     false,
				"CanBeRefunded": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket := Ticket{Status: tt.status}
			
			if got := ticket.IsActive(); got != tt.checks["IsActive"] {
				t.Errorf("Ticket.IsActive() = %v, want %v", got, tt.checks["IsActive"])
			}
			if got := ticket.IsUsed(); got != tt.checks["IsUsed"] {
				t.Errorf("Ticket.IsUsed() = %v, want %v", got, tt.checks["IsUsed"])
			}
			if got := ticket.IsRefunded(); got != tt.checks["IsRefunded"] {
				t.Errorf("Ticket.IsRefunded() = %v, want %v", got, tt.checks["IsRefunded"])
			}
			if got := ticket.CanBeUsed(); got != tt.checks["CanBeUsed"] {
				t.Errorf("Ticket.CanBeUsed() = %v, want %v", got, tt.checks["CanBeUsed"])
			}
			if got := ticket.CanBeRefunded(); got != tt.checks["CanBeRefunded"] {
				t.Errorf("Ticket.CanBeRefunded() = %v, want %v", got, tt.checks["CanBeRefunded"])
			}
		})
	}
}