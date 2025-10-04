package main

type ItemType string

const (
	Digital  ItemType = "digital"
	Physical ItemType = "physical"
)

type Order struct {
	ID       string
	ItemType ItemType
	Amount   int
	InStock  bool
}
