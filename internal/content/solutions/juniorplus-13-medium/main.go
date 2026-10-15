package main

type SlugRepository interface{ Exists(string) bool }

func CanCreateSlug(repo SlugRepository, slug string) bool {
	if repo == nil || len(slug) < 1 || len(slug) > 24 {
		return false
	}
	for _, r := range slug {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return !repo.Exists(slug)
}
