package parcels

import "sort"

type Ref struct {
	Doc    int64   // Document identifier
	Pos    int16   // Position index
	Weight float64 // Weight of the ngram
}

// Check for include item
func refsContains(rs []Ref, r Ref) bool {
	l := len(rs)
	if l == 0 {
		return false
	}

	i := sort.Search(l, func(i int) bool { return rs[i].Doc >= r.Doc })
	if i == l {
		return false
	}

	return rs[i] == r
}

// Append id to the list
func refsInclude(rs []Ref, r Ref) []Ref {
	l := len(rs)
	if l == 0 {
		return []Ref{r}
	}

	i := sort.Search(l, func(i int) bool { return rs[i].Doc >= r.Doc })
	if i == l {
		return append(rs, r)
	}

	if rs[i].Doc == r.Doc {
		return rs
	}

	rs1 := rs[:i]
	rs2 := rs[i:]
	lst := make([]Ref, len(rs1)+len(rs2)+1)
	copy(lst[0:], rs1)
	copy(lst[i+1:], rs2)
	lst[i] = r
	return lst
}

func refsExclude(rs []Ref, id int64) []Ref {
	l := len(rs)
	if l == 0 {
		return nil
	}

	i := sort.Search(l, func(i int) bool { return rs[i].Doc >= id })
	if i == l {
		return rs
	}

	if rs[i].Doc == id {
		if l == 1 {
			return nil
		}
		return append(rs[:i], rs[i+1:]...)
	}

	return rs
}

// Merge two lists
func refsAdd(as, bs []Ref) []Ref {
	la := len(as)
	lb := len(bs)
	if la == 0 {
		return bs
	}
	if lb == 0 {
		return as
	}

	a := 0
	b := 0
	c := make([]Ref, 0, la+lb)
	for a < la && b < lb {
		if as[a].Doc < bs[b].Doc {
			c = append(c, as[a])
			a++
			continue
		}
		if as[a].Doc > bs[b].Doc {
			c = append(c, bs[b])
			b++
			continue
		}
		c = append(c, as[a])
		a++
		b++
	}
	if a < la {
		c = append(c, as[a:]...)
	}
	if b < lb {
		c = append(c, bs[b:]...)
	}
	return c
}

// Subtract list b from list a
func refsSub(as, bs []Ref) []Ref {
	la := len(as)
	lb := len(bs)
	if la == 0 {
		return nil
	}
	if lb == 0 {
		return as
	}

	a := 0
	b := 0
	c := make([]Ref, 0, la)
	for a < la && b < lb {
		if as[a].Doc < bs[b].Doc {
			c = append(c, as[a])
			a++
			continue
		}
		if as[a].Doc > bs[b].Doc {
			b++
			continue
		}
		a++
		b++
	}
	if a < la {
		c = append(c, as[a:]...)
	}
	return c
}
