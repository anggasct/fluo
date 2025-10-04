package main

import (
	"time"
)

type DocumentType string
type Priority string
type ReviewDecision string

const (
	PolicyDoc    DocumentType = "policy"
	TechnicalDoc DocumentType = "technical"
	ContractDoc  DocumentType = "contract"
	GeneralDoc   DocumentType = "general"

	Low      Priority = "low"
	Standard Priority = "standard"
	High     Priority = "high"
	Urgent   Priority = "urgent"

	Approved ReviewDecision = "approved"
	Rejected ReviewDecision = "rejected"
	Pending  ReviewDecision = "pending"
)

type Document struct {
	ID          string
	Title       string
	Author      string
	Type        DocumentType
	Priority    Priority
	Content     string
	CreatedAt   time.Time
	SubmittedAt time.Time
	Version     int
}

type ReviewContext struct {
	Document        *Document
	LegalDecision   ReviewDecision
	LegalReviewer   string
	LegalComments   string
	TechDecision    ReviewDecision
	TechReviewer    string
	TechComments    string
	RejectCount     int
	ReviewStartTime time.Time
	History         []string
	Metadata        map[string]interface{}
}
