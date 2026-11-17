// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"math"
	"testing"

	pb "github.com/open-telemetry/opentelemetry-demo/src/checkoutservice/genproto/oteldemo"
)

func TestMoneyToFloat(t *testing.T) {
	tests := []struct {
		name string
		in   *pb.Money
		want float64
	}{
		{"whole units", &pb.Money{Units: 36}, 36},
		{"cents kept", &pb.Money{Units: 36, Nanos: 550000000}, 36.55},
		{"cents only", &pb.Money{Nanos: 5000000}, 0.005},
		{"negative amount", &pb.Money{Units: -1, Nanos: -750000000}, -1.75},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := moneyToFloat(tt.in)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("moneyToFloat(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
