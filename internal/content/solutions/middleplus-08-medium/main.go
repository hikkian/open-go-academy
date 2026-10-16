package main

func ValidIndexShape(equalityColumns, sortColumns int) bool {
	return equalityColumns >= 0 && sortColumns >= 0 && (equalityColumns > 0 || sortColumns > 0)
}
