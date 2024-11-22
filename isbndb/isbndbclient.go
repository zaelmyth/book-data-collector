package isbndb

type stats struct {
	Books      int
	Authors    int
	Publishers int
	Subjects   int
}

func Stats() stats {
	response := call("/stats")

	return toStruct(response, stats{})
}
