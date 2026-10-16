package main

func PartitionPrunes(queryStart, queryEnd, partStart, partEnd int) bool {
	if queryStart > queryEnd || partStart > partEnd {
		return false
	}
	return queryEnd < partStart || partEnd < queryStart
}
