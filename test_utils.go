package main

func makeCache() MemoryCache {
	return MemoryCache{
		Backend:  make(map[string]*Mesg, 0),
		Maxcount: 0,
	}
}
