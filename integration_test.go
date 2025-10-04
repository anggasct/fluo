package fluo

import (
	"sync"
	"testing"
	"time"
)

func TestIntegration_TrafficLightExample(t *testing.T) {

	definition := NewMachine().
		State("red").Initial().
		To("green").On("timer_expired").
		State("green").
		To("yellow").On("timer_expired").
		State("yellow").
		To("red").On("timer_expired").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()
	AssertState(t, machine, "red")

	states := []string{"green", "yellow", "red", "green", "yellow", "red"}

	for i, expectedState := range states {
		result := machine.HandleEvent("timer_expired", nil)
		AssertEventProcessed(t, result, true)
		AssertState(t, machine, expectedState)

		t.Logf("Cycle %d: transitioned to %s", i+1, expectedState)
	}

	if observer.TransitionCount() != 6 {
		t.Errorf("Expected 6 transitions, got %d", observer.TransitionCount())
	}
}

func TestIntegration_DocumentApprovalWorkflow(t *testing.T) {

	approvalAction := func(ctx Context) error {
		ctx.Set("approved_by", "manager")
		ctx.Set("approval_time", time.Now())
		return nil
	}

	rejectionAction := func(ctx Context) error {
		ctx.Set("rejected_by", "reviewer")
		ctx.Set("rejection_reason", "incomplete")
		return nil
	}

	definition := NewMachine().
		State("draft").Initial().
		To("under_review").On("submit").
		State("under_review").
		To("approved").On("approve").Do(approvalAction).
		To("rejected").On("reject").Do(rejectionAction).
		To("needs_revision").On("request_changes").
		State("needs_revision").
		To("under_review").On("resubmit").
		State("approved").
		To("published").On("publish").
		State("rejected").
		To("draft").On("restart").
		State("published").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()
	machine.Context().Set("document_id", "DOC-123")

	_ = machine.HandleEvent("submit", nil)
	AssertState(t, machine, "under_review")

	_ = machine.HandleEvent("approve", nil)
	AssertState(t, machine, "approved")

	if approvedBy, ok := machine.Context().Get("approved_by"); !ok || approvedBy != "manager" {
		t.Error("Expected approval metadata to be set")
	}

	_ = machine.HandleEvent("publish", nil)
	AssertState(t, machine, "published")

	machine2 := definition.CreateInstance()
	_ = machine2.Start()

	_ = machine2.HandleEvent("submit", nil)
	_ = machine2.HandleEvent("reject", nil)
	AssertState(t, machine2, "rejected")

	if rejectedBy, ok := machine2.Context().Get("rejected_by"); !ok || rejectedBy != "reviewer" {
		t.Error("Expected rejection metadata to be set")
	}

	_ = machine2.HandleEvent("restart", nil)
	AssertState(t, machine2, "draft")

	machine3 := definition.CreateInstance()
	_ = machine3.Start()

	_ = machine3.HandleEvent("submit", nil)
	_ = machine3.HandleEvent("request_changes", nil)
	AssertState(t, machine3, "needs_revision")

	_ = machine3.HandleEvent("resubmit", nil)
	AssertState(t, machine3, "under_review")
}

func TestIntegration_OnlineOrderProcessing(t *testing.T) {

	orderTotal := 0.0

	calculateTotal := func(ctx Context) error {
		if items, ok := ctx.Get("items"); ok {
			orderTotal = items.(float64) * 1.1
			ctx.Set("total", orderTotal)
		}
		return nil
	}

	chargePayment := func(ctx Context) error {
		if total, ok := ctx.Get("total"); ok && total.(float64) > 0 {
			ctx.Set("payment_charged", true)
			ctx.Set("transaction_id", "TXN-12345")
		}
		return nil
	}

	definition := NewMachine().
		State("cart").Initial().
		To("checkout").On("proceed_to_checkout").Do(calculateTotal).
		State("checkout").
		To("payment").On("enter_payment_info").
		State("payment").
		To("processing").On("submit_payment").Do(chargePayment).
		To("cart").On("cancel").
		State("processing").
		To("confirmed").On("payment_approved").
		To("payment_failed").On("payment_declined").
		State("confirmed").
		To("shipped").On("ship_order").
		State("shipped").
		To("delivered").On("delivery_confirmed").
		State("payment_failed").
		To("payment").On("retry_payment").
		State("delivered").
		Build()

	machine := definition.CreateInstance()
	machine.Context().Set("items", 100.0)
	machine.Context().Set("customer_id", "CUST-789")

	_ = machine.Start()
	AssertState(t, machine, "cart")

	_ = machine.HandleEvent("proceed_to_checkout", nil)
	AssertState(t, machine, "checkout")

	if total, ok := machine.Context().Get("total"); !ok {
		t.Error("Expected total to be set in context")
	} else {
		expectedTotal := 110.0
		actualTotal := total.(float64)
		if actualTotal < expectedTotal-0.01 || actualTotal > expectedTotal+0.01 {
			t.Errorf("Expected order total to be approximately %.2f, got %v", expectedTotal, actualTotal)
		}
	}

	_ = machine.HandleEvent("enter_payment_info", nil)
	AssertState(t, machine, "payment")

	_ = machine.HandleEvent("submit_payment", nil)
	AssertState(t, machine, "processing")

	if charged, ok := machine.Context().Get("payment_charged"); !ok || !charged.(bool) {
		t.Error("Expected payment to be charged")
	}

	_ = machine.HandleEvent("payment_approved", nil)
	AssertState(t, machine, "confirmed")

	_ = machine.HandleEvent("ship_order", nil)
	AssertState(t, machine, "shipped")

	_ = machine.HandleEvent("delivery_confirmed", nil)
	AssertState(t, machine, "delivered")
}

func TestIntegration_ServerConnectionManagement(t *testing.T) {

	connectionCount := 0
	maxRetries := 3
	retryCount := 0

	connectAction := func(ctx Context) error {
		connectionCount++
		ctx.Set("connection_id", connectionCount)
		ctx.Set("connected_at", time.Now())
		return nil
	}

	retryGuard := func(ctx Context) bool {
		retryCount++
		return retryCount <= maxRetries
	}

	definition := NewMachine().
		State("disconnected").Initial().
		To("connecting").On("connect").
		State("connecting").
		To("connected").On("connection_established").Do(connectAction).
		To("connection_failed").On("connection_timeout").
		To("disconnected").On("connection_refused").
		State("connected").
		To("disconnected").On("disconnect").
		To("connection_lost").On("connection_dropped").
		State("connection_failed").
		To("connecting").On("retry").When(retryGuard).
		To("disconnected").On("retry").Unless(retryGuard).
		State("connection_lost").
		To("connecting").On("reconnect").
		To("disconnected").On("give_up").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()
	AssertState(t, machine, "disconnected")

	_ = machine.HandleEvent("connect", nil)
	AssertState(t, machine, "connecting")

	_ = machine.HandleEvent("connection_established", nil)
	AssertState(t, machine, "connected")

	if connID, ok := machine.Context().Get("connection_id"); !ok || connID.(int) != 1 {
		t.Error("Expected connection metadata to be set")
	}

	_ = machine.HandleEvent("connection_dropped", nil)
	AssertState(t, machine, "connection_lost")

	_ = machine.HandleEvent("reconnect", nil)
	AssertState(t, machine, "connecting")

	_ = machine.HandleEvent("connection_established", nil)
	AssertState(t, machine, "connected")

	retryCount = 0
	machine2 := definition.CreateInstance()
	_ = machine2.Start()

	_ = machine2.HandleEvent("connect", nil)
	_ = machine2.HandleEvent("connection_timeout", nil)
	AssertState(t, machine2, "connection_failed")

	for i := 1; i <= maxRetries; i++ {
		result := machine2.HandleEvent("retry", nil)
		AssertEventProcessed(t, result, true)
		AssertState(t, machine2, "connecting")

		_ = machine2.HandleEvent("connection_timeout", nil)
		AssertState(t, machine2, "connection_failed")
	}

	result := machine2.HandleEvent("retry", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine2, "disconnected")
}

func TestIntegration_ConcurrentMachines(t *testing.T) {

	definition := NewMachine().
		State("idle").Initial().
		To("working").On("start").
		State("working").
		To("idle").On("finish").
		Build()

	const numMachines = 50
	const eventsPerMachine = 20

	machines := make([]Machine, numMachines)
	observers := make([]*TestObserver, numMachines)

	for i := 0; i < numMachines; i++ {
		machines[i] = definition.CreateInstance()
		observers[i] = NewTestObserver()
		machines[i].AddObserver(observers[i])
		_ = machines[i].Start()
	}

	var wg sync.WaitGroup

	for i := 0; i < numMachines; i++ {
		wg.Add(1)
		go func(machineIndex int) {
			defer wg.Done()
			machine := machines[machineIndex]

			for j := 0; j < eventsPerMachine; j++ {
				eventName := "start"
				if j%2 == 1 {
					eventName = "finish"
				}

				result := machine.HandleEvent(eventName, j)
				if !result.Processed {

					continue
				}
			}
		}(i)
	}

	wg.Wait()

	totalTransitions := 0
	for i, observer := range observers {
		transitions := observer.TransitionCount()
		totalTransitions += transitions

		if transitions == 0 {
			t.Errorf("Machine %d had no transitions", i)
		}

		currentState := machines[i].CurrentState()
		if currentState != "idle" && currentState != "working" {
			t.Errorf("Machine %d in invalid state: %s", i, currentState)
		}
	}

	t.Logf("Total transitions across %d machines: %d", numMachines, totalTransitions)
}

func TestIntegration_LongRunningWorkflow(t *testing.T) {

	batchSize := 100
	processedItems := 0

	processItem := func(ctx Context) error {
		processedItems++
		if processedItems%10 == 0 {
			ctx.Set("progress", processedItems)
		}
		return nil
	}

	checkComplete := func(ctx Context) bool {
		return processedItems >= batchSize
	}

	definition := NewMachine().
		State("idle").Initial().
		To("processing").On("start_batch").
		State("processing").
		To("item_processing").On("process_next").
		State("item_processing").
		To("processing").On("item_complete").Do(processItem).
		State("processing").
		To("completed").On("check_completion").When(checkComplete).
		ToSelf().On("check_completion").Unless(checkComplete).
		State("completed").
		Build()

	machine := definition.CreateInstance()
	machine.Context().Set("batch_size", batchSize)

	_ = machine.Start()
	_ = machine.HandleEvent("start_batch", nil)

	for processedItems < batchSize {
		_ = machine.HandleEvent("process_next", nil)
		_ = machine.HandleEvent("item_complete", nil)

		if processedItems%20 == 0 {
			result := machine.HandleEvent("check_completion", nil)
			if result.StateChanged && machine.CurrentState() == "completed" {
				break
			}
		}
	}

	if machine.CurrentState() != "completed" {
		_ = machine.HandleEvent("check_completion", nil)
	}

	AssertState(t, machine, "completed")

	if processedItems != batchSize {
		t.Errorf("Expected to process %d items, processed %d", batchSize, processedItems)
	}
}

func TestIntegration_StateMachineComposition(t *testing.T) {

	controllerDefinition := NewMachine().
		State("master_idle").Initial().
		To("master_active").On("activate_system").
		State("master_active").
		To("master_idle").On("deactivate_system").
		Build()

	workerDefinition := NewMachine().
		State("worker_stopped").Initial().
		To("worker_running").On("start_work").
		State("worker_running").
		To("worker_stopped").On("stop_work").
		Build()

	controller := controllerDefinition.CreateInstance()
	worker := workerDefinition.CreateInstance()

	_ = controller.Start()
	_ = worker.Start()

	controllerObserver := NewTestObserver()
	controller.AddObserver(controllerObserver)

	_ = controller.HandleEvent("activate_system", nil)
	AssertState(t, controller, "master_active")

	_ = worker.HandleEvent("start_work", nil)
	AssertState(t, worker, "worker_running")

	_ = controller.HandleEvent("deactivate_system", nil)
	AssertState(t, controller, "master_idle")

	_ = worker.HandleEvent("stop_work", nil)
	AssertState(t, worker, "worker_stopped")

	if controllerObserver.TransitionCount() != 2 {
		t.Error("Expected controller to have 2 transitions")
	}
}

func TestIntegration_PerformanceStressTest(t *testing.T) {

	definition := NewMachine().
		State("s1").Initial().
		To("s2").On("e1").
		State("s2").
		To("s3").On("e2").
		State("s3").
		To("s1").On("e3").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	const numEvents = 10000
	events := []string{"e1", "e2", "e3"}

	start := time.Now()
	processedCount := 0

	for i := 0; i < numEvents; i++ {
		eventName := events[i%len(events)]
		result := machine.HandleEvent(eventName, i)
		if result.Processed {
			processedCount++
		}
	}

	duration := time.Since(start)
	eventsPerSecond := float64(processedCount) / duration.Seconds()

	t.Logf("Processed %d/%d events in %v (%.0f events/sec)",
		processedCount, numEvents, duration, eventsPerSecond)

	if eventsPerSecond < 1000 {
		t.Errorf("Performance too low: %.0f events/sec", eventsPerSecond)
	}

	finalState := machine.CurrentState()
	expectedStates := []string{"s1", "s2", "s3"}
	validState := false
	for _, state := range expectedStates {
		if finalState == state {
			validState = true
			break
		}
	}

	if !validState {
		t.Errorf("Machine in unexpected final state: %s", finalState)
	}
}

func TestIntegration_ErrorRecoveryWorkflow(t *testing.T) {

	errorCount := 0
	maxErrors := 3

	maybeFailAction := func(ctx Context) error {
		errorCount++
		if errorCount <= maxErrors {
			originalErr := NewStateError(ErrCodeActionFailed, "processing", "simulated failure")
			return NewActionError("process_action", "processing", originalErr)
		}
		return nil
	}

	definition := NewMachine().
		State("normal").Initial().
		To("processing").On("process").
		State("processing").
		To("success").On("complete").Do(maybeFailAction).
		To("error_handling").On("error").
		State("error_handling").
		To("processing").On("retry").
		To("failed").On("give_up").
		State("success").
		State("failed").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()

	_ = machine.HandleEvent("process", nil)
	AssertState(t, machine, "processing")

	for i := 1; i <= maxErrors; i++ {

		_ = machine.HandleEvent("complete", nil)
		t.Logf("Attempt %d: Error count now %d", i, errorCount)

		if machine.CurrentState() == "success" {

			_ = machine.SetState("processing")
		}

		_ = machine.HandleEvent("error", nil)
		AssertState(t, machine, "error_handling")

		_ = machine.HandleEvent("retry", nil)
		AssertState(t, machine, "processing")
	}

	result := machine.HandleEvent("complete", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "success")

	if machine.CurrentState() != "success" {
		t.Errorf("Expected final state 'success', got '%s'", machine.CurrentState())
	}
}
