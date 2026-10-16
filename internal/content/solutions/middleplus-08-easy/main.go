package main

func IndexOrder(equalityColumns, sortColumns int) string {
	if equalityColumns < 0 || sortColumns < 0 {
		return "invalid"
	}
	if equalityColumns == 0 {
		return "sort_only"
	}
	if sortColumns == 0 {
		return "equality_only"
	}
	return "equality_then_sort"
}
