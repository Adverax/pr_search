package parcels

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestLayout(t *testing.T) {
	type Test struct {
		layout Layout
		src    string
		dst    string
	}

	tests := map[string]Test{
		"1": {
			layout: layoutEn2RuKeyboard,
			src:    "ghbvth ghjcnjuj ntrcnf",
			dst:    "пример простого текста",
		},
		"2": {
			layout: layoutEn2RuKeyboard,
			src:    "fcrjh,byjdfz",
			dst:    "аскорбиновая",
		},
		"3": {
			layout: layoutEn2RuKeyboard,
			src:    "fcrjh,",
			dst:    "аскорб",
		},
		"4": {
			layout: layoutRu2UaPhonetic,
			src:    "аспирин",
			dst:    "аспірін",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dst := string(test.layout.Translate([]rune(test.src)))
			assert.Equal(t, test.dst, dst)
		})
	}
}

/*func TestUnifyNgrams(t *testing.T) {
	type Test struct {
		p, q               []NgramId
		min, max, sequence int
	}

	tests := map[string]Test{
		"1": {
			p:        []NgramId{1, 2, 3, 4},
			q:        []NgramId{3, 4},
			min:      2,
			max:      3,
			sequence: 2,
		},
		"2": {
			p:        []NgramId{3, 4},
			q:        []NgramId{1, 2, 3, 4},
			min:      0,
			max:      1,
			sequence: 2,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			min, max, sequence := unifyNgrams(test.p, test.q)
			assert.Equal(t, test.min, min)
			assert.Equal(t, test.max, max)
			assert.Equal(t, test.sequence, sequence)
		})
	}
}*/

func TestXXX(t *testing.T) {
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	m := make(map[int64]int64, 1000)
	for i := 1; i < 100000; i++ {
		v := rand.Uint64()
		//v := i
		m[int64(v)] = 1
	}

	sz := unsafe.Sizeof(m)
	fmt.Printf("sz %d\n", sz)

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	fmt.Printf("used %d\n", m2.Alloc-m1.Alloc)
}
