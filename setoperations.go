package main

func createTileSet() map[Tile]bool {
	return map[Tile]bool{}
}

func setUnion(s1 map[int]bool, s2 map[int]bool) {
	s_union := map[int]bool{}
	for k := range s1 {
		s_union[k] = true
	}
	for k := range s2 {
		s_union[k] = true
	}
}

func setIntersection(s1 map[int]bool, s2 map[int]bool) {
	s_intersection := map[int]bool{}
	if len(s1) > len(s2) {
		s1, s2 = s2, s1 // better to iterate over a shorter set
	}
	for k := range s1 {
		if s2[k] {
			s_intersection[k] = true
		}
	}
}
