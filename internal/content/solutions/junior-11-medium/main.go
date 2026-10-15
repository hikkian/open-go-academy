package main

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
)

type Item struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

func ReadItems(r io.Reader) ([]Item, error) {
	b, e := io.ReadAll(r)
	if e != nil {
		return nil, e
	}
	var v []Item
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if e = d.Decode(&v); e != nil {
		return nil, e
	}
	var extra any
	if e = d.Decode(&extra); e != io.EOF {
		return nil, errors.New("trailing data")
	}
	for i := range v {
		v[i].Name = strings.TrimSpace(v[i].Name)
		if strings.TrimSpace(v[i].Name) == "" || v[i].Quantity <= 0 {
			return nil, errors.New("invalid item")
		}
	}
	return v, nil
}
