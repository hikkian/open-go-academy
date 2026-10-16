package main

func IndexSupportsQuery(equalityPrefix, indexSortColumns, querySortColumns int, partialPredicateMatches bool) bool {
	return equalityPrefix >= 0 && indexSortColumns >= 0 && querySortColumns >= 0 && (equalityPrefix > 0 || indexSortColumns > 0) && querySortColumns <= indexSortColumns && partialPredicateMatches
}
