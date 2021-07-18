package parcels

import (
	"context"
	"encoding/gob"
	"strings"
)

type NgramEntry struct {
	Id    NgramId // Ngram identifier
	Text  string  // Literal representation
	Score float64 // Ngram score [0..1]
	Pos   int16   // Ngram position
}

type Progress struct {
	Len float32 // Lenght
	Pos float32 // Position [1..Len]
}

type ParserInfo struct {
	Abs Progress
	Rel Progress
}

type ParserEstimator interface {
	Estimate(info *ParserInfo) float64
}

type ParserEstimatorPrimary struct {
	Abs float64
	Rel float64
}

func (e *ParserEstimatorPrimary) Estimate(info *ParserInfo) float64 {
	abs := float64(info.Abs.Pos) / float64(info.Abs.Len)
	rel := float64(info.Rel.Pos) / float64(info.Rel.Len)
	estimation := e.Abs*abs + e.Rel*rel
	// if debug {
	// 	if info.Abs.Pos > info.Abs.Len {
	// 		log.Println("HINT: ESTIMATOR ABS ", info.Abs.Pos, info.Abs.Len)
	// 	}
	// 	if info.Rel.Pos > info.Rel.Len {
	// 		log.Println("HINT: ESTIMATOR REL ", info.Rel.Pos, info.Rel.Len)
	// 	}
	// 	if estimation > 1 {
	// 		log.Println("HINT: ESTIMATION", estimation)
	// 	}
	// }
	return estimation
}

func NewParserEstimatorPrimary(
	abs float64,
) ParserEstimator {
	return &ParserEstimatorPrimary{
		Abs: abs,
		Rel: 1 - abs,
	}
}

type ParserEstimatorSecondary struct {
}

func (e *ParserEstimatorSecondary) Estimate(info *ParserInfo) float64 {
	return float64(info.Abs.Pos) / float64(info.Abs.Len)
}

func NewParserEstimatorSecondary() ParserEstimator {
	return &ParserEstimatorSecondary{}
}

type NgramParser interface {
	// Purge dictionary
	Purge()
	// Parse runes
	Parse(ctx context.Context, runes []rune, allowNew bool) []NgramEntry
	// Find ngram text
	Find(id NgramId) string
	// Get ngram count
	Count() int
}

type NgramParserBase struct {
	Counter   NgramId            // counter for ngrams
	Len       int                // length of ngrams
	Items     map[string]NgramId // dictionary: ngramText -> ngramId
	Estimator ParserEstimator
}

func (parser *NgramParserBase) Find(id NgramId) string {
	for s, v := range parser.Items {
		if v == id {
			return s
		}
	}
	return ""
}

func (parser *NgramParserBase) Purge() {
	parser.Items = map[string]NgramId{}
}

func (parser *NgramParserBase) Count() int {
	return len(parser.Items)
}

func (parser *NgramParserBase) extends(
	ngrams []NgramEntry,
	text string,
	ngram NgramEntry,
	allowNew bool,
) []NgramEntry {
	n, ok := parser.Items[text]
	if !ok {
		if !allowNew {
			return ngrams
		}
		parser.Counter++
		n = parser.Counter
		parser.Items[text] = n
	}

	ngram.Id = n
	return append(ngrams, ngram)
}

// Формирование списка нграм с дроблениме на лексемы
type NgramParserPrimary struct {
	NgramParserBase
}

type chunk struct {
	src int
	dst int
}

func split(s string) (res []chunk) {
	var org, pos int
	lim := len(s)
	for pos < lim {
		if s[pos] == ' ' {
			if org < pos {
				res = append(res, chunk{org, pos})
			}
			org = pos + 1
		}
		pos++
	}
	if org < pos {
		res = append(res, chunk{org, pos})
	}
	return res
}

func (parser *NgramParserPrimary) Parse(
	ctx context.Context,
	runes []rune,
	allowNew bool,
) []NgramEntry {
	runes = SkipPunct(ctx, runes)
	length := len(runes) - parser.Len + 1
	if length <= 0 {
		return nil
	}

	sum := sumN(length)
	info := ParserInfo{
		Abs: Progress{
			Len: float32(sum),
		},
	}
	ngrams := make([]NgramEntry, 0, length)
	source := string(runes)
	chunks := split(source)
	for _, ch := range chunks {
		pos := ch.src
		src := []rune(source[ch.src:ch.dst])
		mlen := len(src)
		info.Rel.Len = float32(mlen)
		if mlen >= parser.Len {
			for p := 0; p <= mlen-parser.Len; p++ {
				ngram := src[p : p+parser.Len]
				info.Abs.Pos = float32(length - pos - p)
				info.Rel.Pos = float32(mlen - p)
				ngrams = parser.extends(
					ngrams,
					string(ngram),
					NgramEntry{
						Pos:   int16(pos + p),
						Text:  string(ngram),
						Score: parser.Estimator.Estimate(&info),
					},
					allowNew,
				)
			}
		}
	}
	return ngrams
}

func NewNgramParserPrimary(
	len int,
	estimator ParserEstimator,
) NgramParser {
	return &NgramParserPrimary{
		NgramParserBase: NgramParserBase{
			Len:       len,
			Items:     make(map[string]NgramId, 32768),
			Estimator: estimator,
		},
	}
}

// Парсер без разбивки на токены.
// Удаляет все пробелы из исходного текста.
type NgramParserSecondary struct {
	NgramParserBase
}

func (parser *NgramParserSecondary) Parse(
	ctx context.Context,
	runes []rune,
	allowNew bool,
) []NgramEntry {
	txt := string(runes)
	txt = strings.Replace(txt, " ", "", -1)
	txt = strings.Replace(txt, "\t", "", -1)
	text := []rune(txt)

	length := len(text) - parser.Len + 1
	if length <= 0 {
		return nil
	}

	sum := sumN(length)
	info := ParserInfo{
		Abs: Progress{
			Len: float32(sum),
		},
		Rel: Progress{
			Len: float32(sum),
		},
	}
	ngrams := make([]NgramEntry, 0, length)
	for p := 0; p < length; p++ {
		ngram := text[p : p+parser.Len]
		info.Abs.Pos = float32(length - p)
		info.Rel.Pos = info.Abs.Pos
		score := parser.Estimator.Estimate(&info)
		ngrams = parser.extends(
			ngrams,
			string(ngram),
			NgramEntry{
				Pos:   int16(p),
				Text:  string(ngram),
				Score: score,
			},
			allowNew,
		)
	}

	return ngrams
}

func NewNgramParserSecondary(
	len int,
	estimator ParserEstimator,
) NgramParser {
	return &NgramParserSecondary{
		NgramParserBase: NgramParserBase{
			Len:       len,
			Items:     make(map[string]NgramId, 32768),
			Estimator: estimator,
		},
	}
}

func init() {
	gob.Register(&NgramParserBase{})
	gob.Register(&NgramParserPrimary{})
	gob.Register(&NgramParserSecondary{})
	gob.Register(&ParserEstimatorPrimary{})
	gob.Register(&ParserEstimatorSecondary{})
}

func sumN(n int) int {
	return sumAB(1, n)
}

func sumAB(a, b int) int {
	return (a + b) * (b - a + 1) / 2
}
