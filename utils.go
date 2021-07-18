package parcels

import (
	"context"
	"strings"
	"unicode/utf8"
)

const (
	AlphabetNum       = "0123456789"
	AlphabetEn        = "abcdefghijklmnopqrstuvwxyz"
	AlphabetRu        = "абвгдеёжзийклмнопрстуфхцчшщъыьэюя"
	AlphabetUa        = "абвгдеєжзиіїйклмнопрстуфхцчшщьюя"
	AlphabetPunct     = `~!@#$%^&*()_-+={}[];:'"|\<>,./?`
	AlphabetSpace     = " "
	AlphabetNonAlpha  = AlphabetNum + AlphabetPunct + AlphabetSpace
	DefaultAlphabetRu = AlphabetEn + AlphabetNum + AlphabetRu + AlphabetSpace
)

type RuneValidator interface {
	IsValid(runes []rune) bool
}

type Dict map[rune]bool

func (dict Dict) IsValid(runes []rune) bool {
	for _, r := range runes {
		if _, ok := dict[r]; !ok {
			if !strings.Contains(AlphabetSpace, string(r)) {
				return false
			}
		}
	}
	return true
}

func newDict(alphabet string) Dict {
	m := make(map[rune]bool)
	for _, r := range []rune(alphabet) {
		m[r] = true
	}
	return m
}

var (
	dictEn    = newDict(AlphabetEn + AlphabetNonAlpha)
	dictRu    = newDict(AlphabetRu + AlphabetNonAlpha)
	dictUa    = newDict(AlphabetUa + AlphabetNonAlpha)
	dictPunct = newDict(AlphabetPunct)

	layoutEn2RuKeyboard = Layout{
		'q':   'й',
		'w':   'ц',
		'e':   'у',
		'r':   'к',
		't':   'е',
		'y':   'н',
		'u':   'г',
		'i':   'ш',
		'o':   'щ',
		'p':   'з',
		'[':   'х',
		']':   'ъ',
		'a':   'ф',
		's':   'ы',
		'd':   'в',
		'f':   'а',
		'g':   'п',
		'h':   'р',
		'j':   'о',
		'k':   'л',
		'l':   'д',
		';':   'ж',
		0x027: 'э',
		'z':   'я',
		'x':   'ч',
		'c':   'с',
		'v':   'м',
		'b':   'и',
		'n':   'т',
		'm':   'ь',
		',':   'б',
		'.':   'ю',
		':':   'ж',
		'"':   'э',
		'<':   'б',
		'>':   'ю',
		'`':   'ё',
		'~':   'ё',
		//`ї`:
		//є
	}
	layoutEn2UaKeyboard = Layout{
		'q':   'й',
		'w':   'ц',
		'e':   'у',
		'r':   'к',
		't':   'е',
		'y':   'н',
		'u':   'г',
		'i':   'ш',
		'o':   'щ',
		'p':   'з',
		'[':   'х',
		']':   'ї',
		'a':   'ф',
		's':   'ы',
		'd':   'в',
		'f':   'а',
		'g':   'п',
		'h':   'р',
		'j':   'о',
		'k':   'л',
		'l':   'д',
		';':   'ж',
		0x027: 'э',
		'z':   'я',
		'x':   'ч',
		'c':   'с',
		'v':   'м',
		'b':   'и',
		'n':   'т',
		'm':   'ь',
		',':   'б',
		'.':   'ю',
		':':   'ж',
		'"':   'є',
		'<':   'б',
		'>':   'ю',
		'`':   'ё',
		'~':   'ё',
	}

	layoutRu2UaKeyboard = Layout{}
	layoutUa2RuKeyboard = Layout{}

	layoutRu2UaPhonetic = Layout{
		'а': 'а',
		'б': 'б',
		'в': 'в',
		'г': 'г',
		'д': 'д',
		'е': 'є',
		'ё': ' ',
		'ж': 'ж',
		'з': 'з',
		'и': 'і',
		'й': 'й',
		'к': 'к',
		'л': 'л',
		'м': 'м',
		'н': 'н',
		'о': 'о',
		'п': 'п',
		'р': 'р',
		'с': 'с',
		'т': 'т',
		'у': 'у',
		'ф': 'ф',
		'х': 'х',
		'ц': 'ц',
		'ч': 'ч',
		'ш': 'ш',
		'щ': 'щ',
		'ъ': ' ',
		'ы': 'и',
		'ь': 'ь',
		'э': 'е',
		'ю': 'ю',
		'я': 'я',
	}

	layoutUa2RuPhonetic = Layout{
		'а': 'а',
		'б': 'б',
		'в': 'в',
		'г': 'г',
		'д': 'д',
		'е': 'э',
		'є': 'е',
		'ж': 'ж',
		'з': 'з',
		'и': 'ы',
		'і': 'и',
		'ї': ' ',
		'й': 'й',
		'к': 'к',
		'л': 'л',
		'м': 'м',
		'н': 'н',
		'о': 'о',
		'п': 'п',
		'р': 'р',
		'с': 'с',
		'т': 'т',
		'у': 'у',
		'ф': 'ф',
		'х': 'х',
		'ц': 'ц',
		'ч': 'ч',
		'ш': 'ш',
		'щ': 'щ',
		'ь': 'ь',
		'ю': 'ю',
		'я': 'я',
	}
)

type Layout map[rune]rune

func (layout Layout) Translate(runes []rune) []rune {
	res := make([]rune, len(runes))
	for i, r := range runes {
		if ch, ok := layout[r]; ok {
			res[i] = ch
		} else {
			res[i] = r
		}
	}

	return res
}

func init() {
	for k, ru := range layoutEn2RuKeyboard {
		if ua, ok := layoutEn2UaKeyboard[k]; ok {
			layoutRu2UaKeyboard[ru] = ua
			layoutUa2RuKeyboard[ua] = ru
		}
	}
}

func NewAlphabetFilter(alphabet string) func(runes []rune) []rune {
	if alphabet == "" {
		alphabet = DefaultAlphabetRu
	}

	m := make(map[rune]bool)
	for _, ch := range alphabet {
		m[ch] = true
	}

	return func(runes []rune) []rune {
		ss := make([]rune, 0, len(runes))
		for _, r := range runes {
			if _, ok := m[r]; ok {
				ss = append(ss, r)
			}
		}

		return ss
	}
}

type mutatorDefault struct {
}

func (m *mutatorDefault) Mute(ctx context.Context, runes []rune) []rune {
	return runes
}

var defaultMutator = new(mutatorDefault)

type predicateDefault struct {
}

func (p *predicateDefault) Test(ctx context.Context, runes []rune) bool {
	return true
}

var defaultPredicate = new(predicateDefault)

func newHypotheses() Hypotheses {
	return make(Hypotheses, 8192)
}

func SkipPunct(ctx context.Context, runes []rune) []rune {
	res := make([]rune, len(runes))
	for i, r := range runes {
		if _, ok := dictPunct[r]; ok {
			res[i] = ' '
		} else {
			res[i] = r
		}
	}

	return res
}

func AllRu(runes []rune) bool {
	return dictRu.IsValid(runes)
}

func AllUa(runes []rune) bool {
	return dictUa.IsValid(runes)
}

var emptyHypotheses = newHypotheses()

// from https://github.com/agnivade/levenshtein
// ComputeDistance computes the levenshtein distance between the two
// strings passed as an argument. The return value is the levenshtein distance
//
// Works on runes (Unicode code points) but does not normalize
// the input strings. See https://blog.golang.org/normalization
// and the golang.org/x/text/unicode/norm pacage.
func ComputeDistance(a, b string) int {
	if len(a) == 0 {
		return utf8.RuneCountInString(b)
	}

	if len(b) == 0 {
		return utf8.RuneCountInString(a)
	}

	if a == b {
		return 0
	}

	// We need to convert to []rune if the strings are non-ascii.
	// This could be avoided by using utf8.RuneCountInString
	// and then doing some juggling with rune indices.
	// The primary challenge is keeping track of the previous rune.
	// With a range loop, its not that easy. And with a for-loop
	// we need to keep track of the inter-rune width using utf8.DecodeRuneInString
	s1 := []rune(a)
	s2 := []rune(b)

	// swap to save some memory O(min(a,b)) instead of O(a)
	if len(s1) > len(s2) {
		s1, s2 = s2, s1
	}
	lenS1 := len(s1)
	lenS2 := len(s2)

	// init the row
	x := make([]int, lenS1+1)
	for i := 0; i < len(x); i++ {
		x[i] = i
	}

	// make a dummy bounds check to prevent the 2 bounds check down below.
	// The one inside the loop is particularly costly.
	_ = x[lenS1]
	// fill in the rest
	for i := 1; i <= lenS2; i++ {
		prev := i
		var current int
		for j := 1; j <= lenS1; j++ {
			if s2[i-1] == s1[j-1] {
				current = x[j-1] // match
			} else {
				current = min(min(x[j-1]+1, prev+1), x[j]+1)
			}
			x[j-1] = prev
			prev = current
		}
		x[lenS1] = prev
	}
	return x[lenS1]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

/*
type HypothesesUnion struct {
	Hypotheses Hypotheses
	Weight     float32
}

func mix(unions []*HypothesesUnion) Hypotheses {
	res := newHypotheses()
	for _, u := range unions {
		for k, v := range u.Hypotheses {
			if vv, ok := res[k]; ok {
				res[k] = vv + v*u.Weight
			} else {
				res[k] = v * u.Weight
			}
		}
	}
	return res
}*/
