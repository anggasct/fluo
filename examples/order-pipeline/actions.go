package main

import (
	"fmt"

	"github.com/anggasct/fluo"
)

func log(message string) fluo.ActionFunc {
	return func(ctx fluo.Context) error {
		fmt.Printf("[LOG] %s\n", message)
		return nil
	}
}

func getOrder(ctx fluo.Context) *Order {
	if v, ok := ctx.Get("order"); ok {
		if o, ok := v.(*Order); ok {
			return o
		}
	}
	return nil
}

func isDigital(ctx fluo.Context) bool {
	if o := getOrder(ctx); o != nil {
		return o.ItemType == Digital
	}
	return false
}

func markPaid(ctx fluo.Context) error {
	ctx.Set("paid", true)
	ctx.Set("payment_ok", true)
	fmt.Println("Payment confirmed")
	return nil
}

func markRiskCleared(ctx fluo.Context) error {
	ctx.Set("risk", "cleared")
	ctx.Set("risk_cleared", true)
	fmt.Println("Risk check cleared")
	return nil
}

func markPacked(ctx fluo.Context) error {
	ctx.Set("packed", true)
	ctx.Set("pack_done", true)
	fmt.Println("Package completed")
	return nil
}

func markLabelReady(ctx fluo.Context) error {
	ctx.Set("label_ready", true)
	ctx.Set("label_done", true)
	fmt.Println("Shipping label ready")
	return nil
}
