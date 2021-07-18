package parcels

import (
	"math"
	"sort"
	"spWebFront/FrontKeeper/infrastructure/core"
	"strings"
)

// Single version
type version struct {
	doc       *Doc
	name      string  // current name of version
	relevance float64 // relevance value
	len       int     // matching length
	group     string  // group name
}

// List of versions, sorted by relevance DESC, name ASC.
type versions []*version

type versionSorter struct {
	versions
	epsilon float64
}

func (sorter *versionSorter) min() float64 {
	var min float64 = 1000000
	for _, v := range sorter.versions {
		if v.relevance < min {
			min = v.relevance
		}
	}
	return min
}

func (sorter *versionSorter) sort() {
	if len(sorter.versions) == 0 {
		return
	}

	for _, v := range sorter.versions {
		v.name = strings.ToLower(v.doc.NameLong)
	}

	sorter.epsilon = 0.01 * float64(sorter.min()) / float64(len(sorter.versions))
	sort.Sort(sorter)
}

func (sorter *versionSorter) isEqual(a, b float64) bool {
	delta := math.Abs(float64(a - b))
	return delta < sorter.epsilon
}

func (sorter *versionSorter) Swap(i, j int) {
	vs := sorter.versions
	vs[i], vs[j] = vs[j], vs[i]
}

func (sorter *versionSorter) Less(i, j int) bool {
	a := sorter.versions[i]
	b := sorter.versions[j]

	if sorter.isEqual(a.relevance, b.relevance) {
		return core.Compare(a.name, b.name) < 0
	}

	return a.relevance > b.relevance
}

func (sorter *versionSorter) Len() int {
	return len(sorter.versions)
}

func sortVersions(vs versions) {
	sorter := &versionSorter{
		versions: vs,
	}
	sorter.sort()
}

type Searcher interface {
	Search(
		ctx context.Context,
		manager Manager,
		query string,
		details *Details,
	) (Hypotheses, error)
}
