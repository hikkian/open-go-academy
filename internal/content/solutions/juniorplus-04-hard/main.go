package main

import "net/url"

func ResourceLocation(alias string) string {
	if alias == "" {
		return ""
	}
	for _, r := range alias {
		if r == ' ' || r == '\t' || r == '\n' {
			return ""
		}
	}
	return "/links/" + url.PathEscape(alias)
}
