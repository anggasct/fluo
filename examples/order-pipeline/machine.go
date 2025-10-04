package main

import (
	"github.com/anggasct/fluo"
)

func BuildOrderMachine() fluo.MachineDefinition {
	b := fluo.NewMachine()

	b.State("order_created").Initial().
		OnEntry(log("Order created")).
		To("prechecks").On("place")

	pre := b.ParallelState("prechecks")
	pay := pre.Region("payment")
	pay.State("pending").Initial().
		OnEntry(log("Waiting for payment confirmation")).
		To("ok").On("pay_ok").Do(markPaid)
	pay.State("ok").Final().
		OnEntry(log("Payment confirmed"))
	risk := pre.Region("risk")
	risk.State("checking").Initial().
		OnEntry(log("Performing risk/fraud checks")).
		To("cleared").On("risk_ok").Do(markRiskCleared)
	risk.State("cleared").Final().
		OnEntry(log("Risk cleared"))
	pre.End()

	b.ParallelState("prechecks").To("digital_fulfillment").OnCompletion().When(func(ctx fluo.Context) bool {
		return isDigital(ctx)
	})
	b.ParallelState("prechecks").To("packaging").OnCompletion().When(func(ctx fluo.Context) bool {
		return !isDigital(ctx)
	})

	b.State("digital_fulfillment").
		OnEntry(log("Fulfilling digital item (send download/email)")).
		To("completed").On("deliver")

	pkg := b.ParallelState("packaging")
	pkr := pkg.Region("pack")
	pkr.State("work").Initial().
		OnEntry(log("Packing items")).
		To("done").On("pack_done").Do(markPacked)
	pkr.State("done").Final().
		OnEntry(log("Package ready"))
	lbr := pkg.Region("label")
	lbr.State("work").Initial().
		OnEntry(log("Creating shipping label")).
		To("done").On("label_done").Do(markLabelReady)
	lbr.State("done").Final().
		OnEntry(log("Label ready"))
	pkg.End()

	b.ParallelState("packaging").To("shipping").OnCompletion()

	b.State("shipping").
		OnEntry(log("Shipping in progress")).
		To("completed").On("ship").
		To("canceled").On("cancel")

	b.State("completed").Final().OnEntry(log("Order completed"))
	b.State("canceled").Final().OnEntry(log("Order canceled"))

	return b.Build()
}
