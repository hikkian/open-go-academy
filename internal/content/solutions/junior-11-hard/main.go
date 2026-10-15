package main

import (
	"bytes"
	"encoding/json"
	"errors"
)

type Item struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

func encodeItems(items []Item) (string, error) {
	for _, v := range items {
		if v.Name == "" || v.Quantity <= 0 {
			return "", errors.New("invalid item")
		}
	}
	var b bytes.Buffer
	e := json.NewEncoder(&b).Encode(items)
	return b.String(), e
}
