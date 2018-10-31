package main

func makeCache() MemoryCache {
	return MemoryCache{
		Backend:  make(map[string]*Mesg, 0),
		Maxcount: 0,
	}
}

func makeQuestionCache(maxCount int) *MemoryQuestionCache {
	return &MemoryQuestionCache{Backend: make([]QuestionCacheEntry, 0), Maxcount: maxCount}
}
