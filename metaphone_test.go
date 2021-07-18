package parcels

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetaphoneRu(t *testing.T) {
	type Test struct {
		src string
		dst string
	}

	tests := map[string]Test{
		"1": {
			src: "швардсенеггер",
			dst: "шварцинигир",
		},
		"2": {
			src: "витавский",
			dst: "витафский",
		},
		"3": {
			src: "витовский",
			dst: "витафский",
		},
		"4": {
			src: "витенберг",
			dst: "витинбирк",
		},
		"5": {
			src: "виттенберг",
			dst: "витинбирк",
		},
		"6": {
			src: "насанов",
			dst: "насанаф",
		},
		"7": {
			src: "насонов",
			dst: "насанаф",
		},
		"8": {
			src: "нассонов",
			dst: "насанаф",
		},
		"9": {
			src: "носонов",
			dst: "насанаф",
		},
		"10": {
			src: "пермаков",
			dst: "пирмакаф",
		},
		"11": {
			src: "пермяков",
			dst: "пирмакаф",
		},
		"12": {
			src: "перьмяков",
			dst: "пирмакаф",
		},
		"13": {
			src: "ополоскиватель",
			dst: "апаласкиватил",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			res := MetaphoneRu([]rune(test.src))
			dst := string(res)
			assert.Equal(t, test.dst, dst)
		})
	}
}
