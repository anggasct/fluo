package main

import "github.com/anggasct/fluo"

func buildDocumentApprovalMachine() fluo.MachineDefinition {
	builder := fluo.NewMachine()

	builder.State("draft").Initial().
		OnEntry(logStateEntry("Document in draft mode")).
		To("submission_routing").On("submit").Do(validateAndSubmitDocument)

	builder.Choice("submission_routing").
		OnEntry(logStateEntry("Routing document based on priority and type")).
		When(isUrgentDocument).To("expedited_review").
		When(isStandardDocument).To("initial_review").
		Otherwise("initial_review")

	builder.State("expedited_review").
		OnEntry(logStateEntry("Expedited review for urgent documents")).
		To("final_decision").On("expedited_complete").Do(processExpeditedReview)

	builder.State("initial_review").
		OnEntry(logStateEntry("Initial document review")).
		To("priority_routing").On("review_complete").Do(initializeReviewProcess)

	builder.Choice("priority_routing").
		OnEntry(logStateEntry("Routing based on document priority")).
		When(isHighPriorityDoc).To("high_priority_path").
		When(isStandardPriorityDoc).To("standard_review_path").
		Otherwise("low_priority_path")

	builder.State("high_priority_path").
		OnEntry(logStateEntry("High priority review path - SLA: 24 hours")).
		To("parallel_fork").On("priority_review_complete").Do(markPriorityReviewComplete)

	builder.State("standard_review_path").
		OnEntry(logStateEntry("Standard review path - SLA: 5 business days")).
		To("parallel_fork").On("standard_review_complete").Do(markStandardReviewComplete)

	builder.State("low_priority_path").
		OnEntry(logStateEntry("Low priority review path - SLA: 10 business days")).
		To("parallel_fork").On("low_priority_complete").Do(markLowPriorityComplete)

	builder.Fork("parallel_fork").
		OnEntry(logStateEntry("Fork: Activating ALL target states simultaneously")).
		To("legal_approval_branch", "technical_approval_branch").
		Do(initiateParallelApproval)

	builder.State("legal_approval_branch").
		OnEntry(logStateEntry("Legal approval branch active (parallel)")).
		To("legal_decision").On("legal_review_done").Do(processLegalReview)

	builder.Choice("legal_decision").
		OnEntry(logStateEntry("Making legal decision")).
		When(isLegalApproved).To("legal_approved").
		Otherwise("legal_rejected")

	builder.State("legal_approved").
		OnEntry(logStateEntry("Legal approval granted")).
		To("sync_join").On("legal_complete").Do(recordLegalApproval)

	builder.State("legal_rejected").
		OnEntry(logStateEntry("Legal approval denied")).
		To("sync_join").On("legal_complete").Do(recordLegalRejection)

	builder.State("technical_approval_branch").
		OnEntry(logStateEntry("Technical approval branch active (parallel)")).
		To("technical_decision").On("technical_review_done").Do(processTechnicalReview)

	builder.Choice("technical_decision").
		OnEntry(logStateEntry("Making technical decision")).
		When(isTechnicalApproved).To("technical_approved").
		Otherwise("technical_rejected")

	builder.State("technical_approved").
		OnEntry(logStateEntry("Technical approval granted")).
		To("sync_join").On("technical_complete").Do(recordTechnicalApproval).
		To("revision_required").On("revise").Do(handleDocumentRevision)

	builder.State("technical_rejected").
		OnEntry(logStateEntry("Technical approval denied")).
		To("sync_join").On("technical_complete").Do(recordTechnicalRejection)

	builder.Join("sync_join").
		OnEntry(logStateEntry("Join: Synchronizing parallel branches")).
		From("legal_approved", "technical_approved").
		From("legal_approved", "technical_rejected").
		From("legal_rejected", "technical_approved").
		From("legal_rejected", "technical_rejected").
		To("consolidation_junction").
		Do(synchronizeApprovals)

	builder.Junction("consolidation_junction").
		OnEntry(logStateEntry("Consolidating parallel results")).
		To("final_decision").
		Do(consolidateReviewResults)

	builder.Fork("fork_demo").
		OnEntry(logStateEntry("Fork demonstration: Splitting to multiple branches")).
		To("branch_a", "branch_b", "branch_c").
		Do(initiateForkDemo)

	builder.State("branch_a").
		OnEntry(logStateEntry("Fork branch A active")).
		To("join_demo").On("branch_a_done").Do(completeBranchA)

	builder.State("branch_b").
		OnEntry(logStateEntry("Fork branch B active")).
		To("join_demo").On("branch_b_done").Do(completeBranchB)

	builder.State("branch_c").
		OnEntry(logStateEntry("Fork branch C active")).
		To("join_demo").On("branch_c_done").Do(completeBranchC)

	builder.Join("join_demo").
		OnEntry(logStateEntry("Join demonstration: Synchronizing all branches")).
		From("branch_a", "branch_b", "branch_c").
		To("fork_demo_complete").
		Do(synchronizeForkBranches)

	builder.State("fork_demo_complete").Final().
		OnEntry(logStateEntry("Fork/Join demonstration completed"))

	builder.History("review_history").
		OnEntry(logStateEntry("Restoring previous review state")).
		Default("initial_review").
		Do(restoreWorkflowState)

	builder.DeepHistory("deep_review_history").
		OnEntry(logStateEntry("Deep restore of workflow state")).
		Default("initial_review").
		Do(deepRestoreWorkflowState)

	builder.Choice("final_decision").
		OnEntry(logStateEntry("Making final approval decision")).
		When(allApprovalsGranted).To("approved").
		When(canResubmitDocument).To("revision_required").
		Otherwise("rejected")

	builder.State("revision_required").
		OnEntry(logStateEntry("Document requires revision")).
		To("draft").On("revise").Do(handleDocumentRevision).
		To("submission_routing").On("submit").Do(validateAndSubmitDocument)

	builder.State("approved").
		OnEntry(logStateEntry("Document approved - workflow complete")).
		To("fork_demo").On("fork_demo_trigger").Do(logStateEntry("Starting fork demo"))

	builder.State("rejected").Final().
		OnEntry(logStateEntry("Document rejected - workflow terminated"))

	return builder.Build()
}
