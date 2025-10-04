package main

import (
	"fmt"
	"time"

	"github.com/anggasct/fluo"
)

func isUrgentDocument(ctx fluo.Context) bool {
	if reviewCtx := getReviewContext(ctx); reviewCtx != nil {
		return reviewCtx.Document.Priority == Urgent
	}
	return false
}

func isStandardDocument(ctx fluo.Context) bool {
	if reviewCtx := getReviewContext(ctx); reviewCtx != nil {
		priority := reviewCtx.Document.Priority
		return priority == Standard || priority == High
	}
	return false
}

func isHighPriorityDoc(ctx fluo.Context) bool {
	if reviewCtx := getReviewContext(ctx); reviewCtx != nil {
		return reviewCtx.Document.Priority == High
	}
	return false
}

func isStandardPriorityDoc(ctx fluo.Context) bool {
	if reviewCtx := getReviewContext(ctx); reviewCtx != nil {
		return reviewCtx.Document.Priority == Standard
	}
	return false
}

func isLegalApproved(ctx fluo.Context) bool {
	if reviewCtx := getReviewContext(ctx); reviewCtx != nil {
		return reviewCtx.LegalDecision == Approved
	}
	return false
}

func isTechnicalApproved(ctx fluo.Context) bool {
	if reviewCtx := getReviewContext(ctx); reviewCtx != nil {
		return reviewCtx.TechDecision == Approved
	}
	return false
}

func allApprovalsGranted(ctx fluo.Context) bool {
	if reviewCtx := getReviewContext(ctx); reviewCtx != nil {
		return reviewCtx.LegalDecision == Approved && reviewCtx.TechDecision == Approved
	}
	return false
}

func canResubmitDocument(ctx fluo.Context) bool {
	if reviewCtx := getReviewContext(ctx); reviewCtx != nil {
		return reviewCtx.RejectCount < 3
	}
	return false
}

func validateAndSubmitDocument(ctx fluo.Context) error {
	reviewCtx := getReviewContext(ctx)
	if reviewCtx == nil {
		return fmt.Errorf("invalid review context")
	}

	doc := reviewCtx.Document
	doc.SubmittedAt = time.Now()

	fmt.Printf("Document '%s' submitted by %s\n", doc.Title, doc.Author)
	fmt.Printf("Priority: %s, Type: %s\n", doc.Priority, doc.Type)

	reviewCtx.History = append(reviewCtx.History, fmt.Sprintf("Submitted at %v", doc.SubmittedAt))
	return nil
}

func processExpeditedReview(ctx fluo.Context) error {
	fmt.Println("Processing expedited review - fast track approval")
	reviewCtx := getReviewContext(ctx)
	if reviewCtx != nil {
		reviewCtx.LegalDecision = Approved
		reviewCtx.TechDecision = Approved
		reviewCtx.History = append(reviewCtx.History, "Expedited review completed")
	}
	return nil
}

func initializeReviewProcess(ctx fluo.Context) error {
	reviewCtx := getReviewContext(ctx)
	if reviewCtx == nil {
		return fmt.Errorf("invalid review context")
	}

	reviewCtx.ReviewStartTime = time.Now()
	fmt.Printf("Initializing review process for document: %s\n", reviewCtx.Document.Title)
	reviewCtx.History = append(reviewCtx.History, "Review process initialized")
	return nil
}

func markPriorityReviewComplete(ctx fluo.Context) error {
	fmt.Println("High priority review completed")
	return recordReviewStep(ctx, "High priority review completed")
}

func markStandardReviewComplete(ctx fluo.Context) error {
	fmt.Println("Standard review completed")
	return recordReviewStep(ctx, "Standard review completed")
}

func markLowPriorityComplete(ctx fluo.Context) error {
	fmt.Println("Low priority review completed")
	return recordReviewStep(ctx, "Low priority review completed")
}

func initiateParallelApproval(ctx fluo.Context) error {
	fmt.Println("Initiating parallel legal and technical approval")
	return recordReviewStep(ctx, "Parallel approval initiated")
}

func processLegalReview(ctx fluo.Context) error {
	reviewCtx := getReviewContext(ctx)
	if reviewCtx != nil {
		// Only set decision if it's still pending (respect initial context)
		if reviewCtx.LegalDecision == Pending {
			if reviewCtx.Document.Type == ContractDoc {
				reviewCtx.LegalDecision = Approved
				reviewCtx.LegalComments = "Contract terms acceptable"
			} else {
				reviewCtx.LegalDecision = Approved
				reviewCtx.LegalComments = "Legal review completed successfully"
			}
		}
		// If already set (e.g., to Rejected), keep it as is
		reviewCtx.LegalReviewer = "Legal Team Lead"
		fmt.Printf("Legal review decision: %s\n", reviewCtx.LegalDecision)
	}
	return recordReviewStep(ctx, "Legal review processed")
}

func processTechnicalReview(ctx fluo.Context) error {
	reviewCtx := getReviewContext(ctx)
	if reviewCtx != nil {
		// Only set decision if it's still pending (respect initial context)
		if reviewCtx.TechDecision == Pending {
			if reviewCtx.Document.Type == TechnicalDoc {
				reviewCtx.TechDecision = Approved
				reviewCtx.TechComments = "Technical implementation is sound"
			} else {
				reviewCtx.TechDecision = Approved
				reviewCtx.TechComments = "Technical review completed"
			}
		}
		// If already set (e.g., to Rejected), keep it as is
		reviewCtx.TechReviewer = "Senior Technical Architect"
		fmt.Printf("Technical review decision: %s\n", reviewCtx.TechDecision)
	}
	return recordReviewStep(ctx, "Technical review processed")
}

func recordLegalApproval(ctx fluo.Context) error {
	fmt.Println("Recording legal approval")
	return recordReviewStep(ctx, "Legal approval recorded")
}

func recordLegalRejection(ctx fluo.Context) error {
	fmt.Println("Recording legal rejection")
	return recordReviewStep(ctx, "Legal rejection recorded")
}

func recordTechnicalApproval(ctx fluo.Context) error {
	fmt.Println("Recording technical approval")
	return recordReviewStep(ctx, "Technical approval recorded")
}

func recordTechnicalRejection(ctx fluo.Context) error {
	fmt.Println("Recording technical rejection")
	return recordReviewStep(ctx, "Technical rejection recorded")
}

func synchronizeApprovals(ctx fluo.Context) error {
	reviewCtx := getReviewContext(ctx)
	if reviewCtx != nil {
		fmt.Printf("Synchronizing approvals - Legal: %s, Technical: %s\n",
			reviewCtx.LegalDecision, reviewCtx.TechDecision)
	}
	return recordReviewStep(ctx, "Approvals synchronized")
}

func consolidateReviewResults(ctx fluo.Context) error {
	fmt.Println("Consolidating all review results")
	return recordReviewStep(ctx, "Review results consolidated")
}

func restoreWorkflowState(ctx fluo.Context) error {
	fmt.Println("Restoring previous workflow state from history")
	return recordReviewStep(ctx, "Workflow state restored")
}

func deepRestoreWorkflowState(ctx fluo.Context) error {
	fmt.Println("Deep restoring complex workflow state")
	return recordReviewStep(ctx, "Deep workflow state restored")
}

func handleDocumentRevision(ctx fluo.Context) error {
	reviewCtx := getReviewContext(ctx)
	if reviewCtx != nil {
		reviewCtx.RejectCount++
		reviewCtx.Document.Version++
		fmt.Printf("Document revision required - Version %d, Attempt %d\n",
			reviewCtx.Document.Version, reviewCtx.RejectCount)
	}
	return recordReviewStep(ctx, "Document revision initiated")
}

func initiateForkDemo(ctx fluo.Context) error {
	fmt.Println("Initiating Fork demonstration with parallel execution")
	return recordReviewStep(ctx, "Fork demo initiated")
}

func completeBranchA(ctx fluo.Context) error {
	fmt.Println("Completing Fork branch A")
	return recordReviewStep(ctx, "Branch A completed")
}

func completeBranchB(ctx fluo.Context) error {
	fmt.Println("Completing Fork branch B")
	return recordReviewStep(ctx, "Branch B completed")
}

func completeBranchC(ctx fluo.Context) error {
	fmt.Println("Completing Fork branch C")
	return recordReviewStep(ctx, "Branch C completed")
}

func synchronizeForkBranches(ctx fluo.Context) error {
	fmt.Println("Synchronizing all fork branches with Join logic")
	return recordReviewStep(ctx, "Fork branches synchronized")
}

func logStateEntry(message string) fluo.ActionFunc {
	return func(ctx fluo.Context) error {
		fmt.Printf("%s\n", message)
		return nil
	}
}

func recordReviewStep(ctx fluo.Context, step string) error {
	reviewCtx := getReviewContext(ctx)
	if reviewCtx != nil {
		reviewCtx.History = append(reviewCtx.History, fmt.Sprintf("%s at %v", step, time.Now().Format("15:04:05")))
	}
	return nil
}

func getReviewContext(ctx fluo.Context) *ReviewContext {
	if val, exists := ctx.Get("review_context"); exists {
		if reviewCtx, ok := val.(*ReviewContext); ok {
			return reviewCtx
		}
	}
	return nil
}
