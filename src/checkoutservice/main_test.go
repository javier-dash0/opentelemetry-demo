// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"testing"

	pb "github.com/open-telemetry/opentelemetry-demo/src/checkoutservice/genproto/oteldemo"
)

func TestMultiplyItemCostRejectsQuantityBelowOne(t *testing.T) {
	cost := &pb.Money{CurrencyCode: "USD", Units: 10, Nanos: 0}
	for _, quantity := range []int32{0, -1, -2147483648} {
		item := &pb.OrderItem{
			Item: &pb.CartItem{ProductId: "OLJCESPC7Z", Quantity: quantity},
			Cost: cost,
		}
		got, err := multiplyItemCost(item)
		if err == nil {
			t.Fatalf("quantity %d: expected an error, got %v", quantity, got)
		}
		if got != nil {
			t.Fatalf("quantity %d: expected no money value, got %v", quantity, got)
		}
	}
}

func TestMultiplyItemCostMultipliesCostByQuantity(t *testing.T) {
	item := &pb.OrderItem{
		Item: &pb.CartItem{ProductId: "OLJCESPC7Z", Quantity: 3},
		Cost: &pb.Money{CurrencyCode: "USD", Units: 10, Nanos: 500000000},
	}
	got, err := multiplyItemCost(item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := &pb.Money{CurrencyCode: "USD", Units: 31, Nanos: 500000000}
	if got.GetUnits() != want.GetUnits() || got.GetNanos() != want.GetNanos() || got.GetCurrencyCode() != want.GetCurrencyCode() {
		t.Fatalf("multiplyItemCost() = %v, want %v", got, want)
	}
}
