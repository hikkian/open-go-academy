package main

func ChangeCompatibility(additive, removesField, changesMeaning bool) string {
	if removesField || changesMeaning {
		return "breaking"
	}
	if additive {
		return "candidate"
	}
	return "no_change"
}
