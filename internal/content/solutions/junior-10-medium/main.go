package main

import (
	"errors"
	"strconv"
	"strings"
)

type Config struct {
	Name  string
	Limit int
}

func ParseArgs(args []string) (Config, error) {
	c := Config{}
	seen := map[string]bool{}
	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return Config{}, errors.New("missing value")
		}
		k, v := args[i], args[i+1]
		if seen[k] {
			return Config{}, errors.New("duplicate")
		}
		seen[k] = true
		switch k {
		case "--name":
			c.Name = v
		case "--limit":
			n, e := strconv.Atoi(v)
			if e != nil {
				return Config{}, e
			}
			c.Limit = n
		default:
			return Config{}, errors.New("unknown")
		}
	}
	if strings.TrimSpace(c.Name) == "" || c.Limit <= 0 {
		return Config{}, errors.New("invalid")
	}
	return c, nil
}
