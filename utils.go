package main

func makeCache() MemoryCache {
	return MemoryCache{
		Backend:  make(map[string]*Mesg),
		Maxcount: 0,
	}
}

func makeQuestionCache(maxCount int) *MemoryQuestionCache {
	return &MemoryQuestionCache{Backend: make([]QuestionCacheEntry, 0), Maxcount: maxCount}
}
