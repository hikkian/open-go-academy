package main

import (
	"strings"
)

type Item struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

func isValidItem(item Item) bool { return strings.TrimSpace(item.Name) != "" && item.Quantity > 0 }
