package parcels

import (
	"context"
	"encoding/gob"
	"fmt"
	"math"
	"sort"
	"spWebFront/FrontKeeper/infrastructure/log"
	"strings"
)

// https://habr.com/en/post/114997/
// https://habr.com/en/post/346618/
// https://habr.com/en/company/mailru/blog/267469/
// https://habr.com/en/company/yandex/blog/198556/
// https://play.golang.org/p/ItZqDY6uy5

// Identifier is abstract identifier
type Identifier struct {
	NameVal string
}

// Name is getter for reading same property /
func (id *Identifier) Name() string {
	return id.NameVal
}

// Derivative is abstract derivative rule.
type Derivative struct {
	Identifier
	Rule
}

// Name is getter for reading same property.
func (d *Derivative) Name() string {
	return d.NameVal
}

// MultiRule is abstract rule for combine multiple entries.
type MultiRule struct {
	Identifier
	Entries   Entries
	Estimator Estimator
	Mixer     Mixer
}

func (rule *MultiRule) Clone() Rule {
	return NewMultiRule(
		rule.Identifier.NameVal,
		rule.Entries.Clone(),
		rule.Estimator,
		rule.Mixer,
	)
}

func (rule *MultiRule) Search(
	ctx context.Context,
	resolver Resolver,
	query []rune,
	weight float64,
	details *Details,
) Hypotheses {
	var bs []branch
	for _, e := range rule.Entries {
		h := resolver.Resolve(ctx, e, query, weight, details)
		if len(h) == 0 {
			continue
		}
		w := details.getWeight(e.Name(), e.Weight)
		h1 := h.scale(w)
		bs = append(
			bs,
			branch{
				hypotheses: h1,
				relevance:  rule.Estimator.Estimate(h1),
			},
		)
	}
	return rule.Mixer.Mix(bs)
}

func (rule *MultiRule) Log() {
	for _, e := range rule.Entries {
		e.Log()
	}
}

func (rule *MultiRule) Purge(
	ctx context.Context,
) error {
	for _, e := range rule.Entries {
		err := e.Purge(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rule *MultiRule) Append(
	ctx context.Context,
	id int64,
	name []rune,
	weight float64,
) {
	for _, e := range rule.Entries {
		e.Append(ctx, id, name, weight)
	}
}

func (rule *MultiRule) Remove(
	ctx context.Context,
	id int64,
) {
	for _, e := range rule.Entries {
		e.Remove(ctx, id)
	}
}

// NewMultiRule is constructor for creating new instance of MultiRule
func NewMultiRule(
	name string,
	entries Entries,
	estimator Estimator,
	mixer Mixer,
) Rule {
	return &MultiRule{
		Identifier: Identifier{
			NameVal: name,
		},
		Entries:   entries,
		Estimator: estimator,
		Mixer:     mixer,
	}
}

// GuardRule is rule for check pair of predicates before execute rule.
type GuardRule struct {
	Derivative
	PredicateBuild  Predicate
	PredicateSearch Predicate
}

func (rule *GuardRule) Clone() Rule {
	return NewGuardRule(
		rule.Derivative.NameVal,
		rule.PredicateBuild,
		rule.PredicateSearch,
		rule.Derivative.Rule,
	)
}

func (rule *GuardRule) Append(
	ctx context.Context,
	id int64,
	name []rune,
	weight float64,
) {
	if rule.PredicateBuild.Test(ctx, name) {
		rule.Rule.Append(ctx, id, name, weight)
	}
}

func (rule *GuardRule) Remove(
	ctx context.Context,
	id int64,
) {
	rule.Rule.Remove(ctx, id)
}

func (rule *GuardRule) Search(
	ctx context.Context,
	resolver Resolver,
	query []rune,
	weight float64,
	details *Details,
) Hypotheses {
	if rule.PredicateSearch.Test(ctx, query) {
		return resolver.Resolve(ctx, rule.Rule, query, weight, details)
	}

	return newHypotheses()
}

func NewGuardRule(
	name string,
	predicateBuild Predicate,
	predicateSearch Predicate,
	rule Rule,
) Rule {
	if predicateBuild == nil {
		predicateBuild = defaultPredicate
	}

	if predicateSearch == nil {
		predicateSearch = defaultPredicate
	}

	return &GuardRule{
		Derivative: Derivative{
			Identifier: Identifier{
				NameVal: name,
			},
			Rule: rule,
		},
		PredicateBuild:  predicateBuild,
		PredicateSearch: predicateSearch,
	}
}

// MuteRule is rule, that make distorts to the runes.
type muteRule struct {
	Derivative
	MutatorBuild  Mutator // Distorter func for append/remove operations
	MutatorSearch Mutator // Distorte func for search operation
}

func (rule *muteRule) Clone() Rule {
	return NewMuteRule(
		rule.Derivative.NameVal,
		rule.MutatorBuild,
		rule.MutatorSearch,
		rule.Derivative.Rule,
	)
}

func (rule *muteRule) Append(
	ctx context.Context,
	id int64,
	name []rune,
	weight float64,
) {
	name2 := rule.MutatorBuild.Mute(ctx, name)
	ww := calcWeight(string(name), string(name2))
	rule.Rule.Append(ctx, id, name2, weight*ww)
}

func (rule *muteRule) Remove(
	ctx context.Context,
	id int64,
) {
	rule.Rule.Remove(ctx, id)
}

func (rule *muteRule) Search(
	ctx context.Context,
	resolver Resolver,
	query []rune,
	weight float64,
	details *Details,
) Hypotheses {
	query2 := rule.MutatorSearch.Mute(ctx, query)
	return resolver.Resolve(ctx, rule.Rule, query2, weight, details)
}

// NewMuteRule is constructor for creating instance of MuteRule.
func NewMuteRule(
	name string,
	mutatorBuild Mutator,
	mutatorSearch Mutator,
	rule Rule,
) Rule {
	if mutatorBuild == nil {
		mutatorBuild = defaultMutator
	}

	if mutatorSearch == nil {
		mutatorSearch = defaultMutator
	}

	return &muteRule{
		Derivative: Derivative{
			Identifier: Identifier{
				NameVal: name,
			},
			Rule: rule,
		},
		MutatorBuild:  mutatorBuild,
		MutatorSearch: mutatorSearch,
	}
}

// NgramId is internal ngram identifier
type NgramId uint16

// NgramIndexStatisctics is statistics for ngram index
type NgramIndexStatisctics struct {
	Count int     // Document count
	Refs  int     // Summary references count
	MaxId NgramId // Identifier with max V
	MaxV  int     // Max value
}

// NgramIndex is abstract ngram index
type NgramIndex interface {
	Clone() NgramIndex
	// Purge index
	Purge()
	// Append document
	Append(id int64, ngrams []NgramEntry, weight float64)
	// Remove document
	Remove(id int64)
	// Get statistics
	Statistics() NgramIndexStatisctics
	// Search and append new hypotheses
	Search(query []NgramEntry, weight float64) Hypotheses
}

// NgramIndexPositions is position infor for ngram index
type NgramIndexPositions struct {
	Weight    float64 // Summary weight [0..1]
	Query     float64 // Weight of query [0..1]
	Pattern   float64 // Weight of pattern [0..1]
	Inflation float64 // Speed of inflation [0..1]
}

type ngramIndex struct {
	Items    map[NgramId][]Ref
	Position NgramIndexPositions
	docs     DocManager
}

func (index *ngramIndex) Clone() NgramIndex {
	return NewNgramIndex(index.docs, index.Position)
}

func (index *ngramIndex) Purge() {
	index.Items = make(map[NgramId][]Ref)
}

func (index *ngramIndex) Statistics() (res NgramIndexStatisctics) {
	docs := make(map[int64]bool, 16384)
	for n, item := range index.Items {
		res.Refs += len(item)
		if len(item) > res.MaxV {
			res.MaxV = len(item)
			res.MaxId = n
		}
		for _, doc := range item {
			docs[doc.Doc] = true
		}
	}
	res.Count = len(docs)
	return
}

func (index *ngramIndex) Append(
	id int64,
	ngrams []NgramEntry,
	weight float64,
) {
	for _, n := range ngrams {
		index.append(
			n.Id,
			Ref{
				Doc:    id,
				Pos:    n.Pos,
				Weight: weight,
			},
		)
	}
}

func (index *ngramIndex) Remove(id int64) {
	res := make(map[NgramId][]Ref, len(index.Items))
	for key, refs := range index.Items {
		res[key] = refsExclude(refs, id)
	}
	index.Items = res
}

func (index *ngramIndex) append(ngram NgramId, ref Ref) {
	if rs, ok := index.Items[ngram]; ok {
		index.Items[ngram] = refsInclude(rs, ref)
	} else {
		index.Items[ngram] = []Ref{ref}
	}
}

func (index *ngramIndex) remove(ngram NgramId, doc int64) {
	if rs, ok := index.Items[ngram]; ok {
		l := len(rs)
		i := sort.Search(l, func(i int) bool { return rs[i].Doc >= doc })
		if i < l {
			if len(rs) == 1 {
				delete(index.Items, ngram)
			} else {
				index.Items[ngram] = append(rs[:i], rs[i+1:]...)
			}
		}
	}
}

func (index *ngramIndex) Search(
	query []NgramEntry,
	weight float64,
) Hypotheses {
	if len(query) == 0 {
		return newHypotheses()
	}

	type Raw struct {
		pos float64
		rel float64
	}
	raw := make(map[int64]*Raw, 8192)
	rel := 1 / float64(len(query))
	for _, ngram := range query {
		if rs, ok := index.Items[ngram.Id]; ok {
			for _, r := range rs {
				pos := index.Position.Pattern*float64(r.Pos) + index.Position.Query*float64(ngram.Pos)
				rl := rel * (0.5*r.Weight + 0.5*weight)
				if rr, ok := raw[r.Doc]; ok {
					rr.rel += rl
					rr.pos += pos
				} else {
					raw[r.Doc] = &Raw{
						pos: pos,
						rel: rl,
					}
				}
			}
		}
	}

	hs := newHypotheses()
	posWeight := index.Position.Weight
	for doc, r := range raw {
		sss := math.Pow(10, float64(index.Position.Inflation))
		pos := 1 / math.Pow(sss, float64(r.pos))
		rel := r.rel*(1-posWeight) + pos*posWeight
		hs[doc] = rel
	}

	return hs
}

// NewNgramIndex is constructor for creating instance of ngram index.
func NewNgramIndex(
	docs DocManager,
	positions NgramIndexPositions,
) NgramIndex {
	positions.Pattern = 1 - positions.Query
	return &ngramIndex{
		Items:    make(map[NgramId][]Ref, 32768),
		Position: positions,
	}
}

// NgramRule is rule for ngram search.
type ngramRule struct {
	Identifier
	Parser NgramParser // ngrams
	Index  NgramIndex  // backward index
}

func (rule *ngramRule) Clone() Rule {
	return NewNgramRule(
		rule.Identifier.NameVal,
		rule.Index.Clone(),
		rule.Parser,
	)
}

func (rule *ngramRule) Log() {
	log.DebugFunc(func() {
		stats := rule.Index.Statistics()
		s := rule.Parser.Find(stats.MaxId)
		log.Printf(
			"STATISTICS FOR NGRAM RULE %s: dictionary size = %d, document count = %d, entry count = %d, Ngram max = (%s, %d)",
			rule.NameVal,
			rule.Parser.Count(),
			stats.Count,
			stats.Refs,
			s,
			stats.MaxV,
		)
	})
}

func (rule *ngramRule) Purge(ctx context.Context) error {
	rule.Parser.Purge()
	rule.Index.Purge()
	return nil
}

func (rule *ngramRule) Append(
	ctx context.Context,
	id int64,
	name []rune,
	weight float64,
) {
	ngrams := rule.Parser.Parse(ctx, name, true)
	rule.Index.Append(id, ngrams, weight)
}

func (rule *ngramRule) Remove(
	ctx context.Context,
	id int64,
) {
	rule.Index.Remove(id)
}

func (rule *ngramRule) Search(
	ctx context.Context,
	resolver Resolver,
	query []rune,
	weight float64,
	details *Details,
) Hypotheses {
	ngrams := rule.Parser.Parse(ctx, query, false)
	if debug {
		var lst []string
		for _, n := range ngrams {
			lst = append(lst, fmt.Sprintf("%q=%g", n.Text, n.Score))
		}
		log.Debugf("SEARCH BY NGRAM RULE %q FOR QUERY %q HAS NGRAMS=%d {%s}", rule.NameVal, string(query), len(ngrams), strings.Join(lst, ", "))
	}

	return rule.Index.Search(ngrams, weight)
}

// NewNgramRule is constructor for creating instance of NgramRule
func NewNgramRule(
	name string,
	index NgramIndex,
	parser NgramParser,
) Rule {
	return &ngramRule{
		Identifier: Identifier{
			NameVal: name,
		},
		Index:  index,
		Parser: parser,
	}
}

func indexOfNgram(ns []NgramId, n NgramId) int {
	for i, v := range ns {
		if v == n {
			return i
		}
	}
	return -1
}

// Формирование общей оценки на основании позиций составляющих ngram.
func ngramsEstimate(doc, query []NgramId) float32 {
	n := len(doc)
	if n == 0 {
		return 0
	}
	var res float32
	for i, q := range query {
		index := indexOfNgram(doc, q)
		if index == -1 {
			continue
		}
		res += float32(n - i)
	}
	return res / float32(n)
}

// LayoutTranslator is abstract rune translator at runtime.
type LayoutTranslator interface {
	Translate(ctx context.Context, runes []rune, details *Details) []rune
}

// LayoutValisator is abstract guard for LayoutTransaltor.
type LayoutValidator interface {
	IsValid(ctx context.Context, runes []rune) bool
}

type layoutValidator struct {
	Validator RuneValidator
	Language  string
}

func (validator *layoutValidator) IsValid(ctx context.Context, runes []rune) bool {
	language, _ := ctx.Value("language").(string)
	return ((language == "") || (language == validator.Language)) && validator.Validator.IsValid(runes)
}

// NewLayoutValidator is constructor for creating instance of LayoutValidator.
func NewLayoutValidator(
	validator RuneValidator,
	language string,
) LayoutValidator {
	return &layoutValidator{
		Validator: validator,
		Language:  language,
	}
}

type layoutTranslator struct {
	Layout    Layout
	Validator LayoutValidator
}

func (tr *layoutTranslator) Translate(
	ctx context.Context,
	runes []rune,
	details *Details,
) []rune {
	if !(tr.Validator != nil && tr.Validator.IsValid(ctx, runes)) {
		return nil
	}
	return tr.Layout.Translate(runes)
}

// NewLayoutTranslator is constructor for create instance of LayoutTranslator.
func NewLayoutTranslator(
	layout Layout,
	validator RuneValidator,
	language string,
) LayoutTranslator {
	return &layoutTranslator{
		Layout:    layout,
		Validator: NewLayoutValidator(validator, language),
	}
}

// LayoutRule is rule, that attempts select best layout at the search time.
type layoutRule struct {
	Derivative
	Layouts   []LayoutTranslator
	Weight    float64
	Estimator Estimator
}

func (rule *layoutRule) Search(
	ctx context.Context,
	resolver Resolver,
	query []rune,
	weight float64,
	details *Details,
) Hypotheses {
	hs := make([]Hypotheses, 0, len(rule.Layouts)+1)

	// Выполняем поиск без преобразования символов
	if h := resolver.Resolve(ctx, rule.Rule, query, weight, details); h != nil {
		hs = append(hs, h)
	}

	// Выполняем поиск для каждой из схем преобразования символов
	for _, layout := range rule.Layouts {
		q := layout.Translate(ctx, query, details)
		if q == nil {
			continue
		}
		w := calcWeight(string(query), string(q))
		if h := resolver.Resolve(ctx, rule.Rule, q, weight*w, details); h != nil {
			h = h.scale(rule.Weight)
			hs = append(hs, h)
		}
	}

	// Выполняем поиск лучшего набора гипотез
	br := float64(0)
	var bh Hypotheses
	for _, h := range hs {
		r := rule.Estimator.Estimate(h)
		if br < r {
			br = r
			bh = h
		}
	}

	return bh
}

// NewLayoutRule is constructor for creating instance layout level rule.
func NewLayoutRule(
	name string,
	rule Rule,
	layouts []LayoutTranslator,
	weight float64,
	estimator Estimator,
) Rule {
	return &layoutRule{
		Derivative: Derivative{
			Identifier: Identifier{
				NameVal: name,
			},
			Rule: rule,
		},
		Layouts:   layouts,
		Weight:    weight,
		Estimator: estimator,
	}
}

// FilterRule is rule for apply custom filter at runtime
type filterRule struct {
	Derivative
}

func (rule *filterRule) Search(
	ctx context.Context,
	resolver Resolver,
	query []rune,
	weight float64,
	details *Details,
) Hypotheses {
	hs := resolver.Resolve(ctx, rule.Rule, query, weight, details)
	return details.filter(hs)
}

// NewFilterRule is constructor for create instance of FilterRule.
func NewFilterRule(
	name string,
	rule Rule,
) Rule {
	return &filterRule{
		Derivative: Derivative{
			Identifier: Identifier{
				NameVal: name,
			},
			Rule: rule,
		},
	}
}

func init() {
	gob.Register(Dict{})
	gob.Register(&Identifier{})
	gob.Register(&Derivative{})
	gob.Register(&MultiRule{})
	gob.Register(&GuardRule{})
	gob.Register(&ngramIndex{})
	gob.Register(&ngramRule{})
	gob.Register(&muteRule{})
	gob.Register(&layoutRule{})
	gob.Register(&filterRule{})
	gob.Register(&Ref{})
	gob.Register(&NgramEntry{})
	gob.Register(&layoutValidator{})
	gob.Register(&layoutTranslator{})
}

func calcWeight(a, b string) float64 {
	d := ComputeDistance(a, b)
	return 1 / float64(1+math.Log(float64(d+1)))
}

/*

// Область видимости
type DrugScope interface {
	// Search by ngrams
	Search(ctx context.Context, query []string, limit int) ([]*Ref, error)
	// Refresh search index for single document
	Refresh(ctx context.Context, doc int64, Refs []Ref) (int64, error)

	// Accept ngram
	AcceptNgram(ctx context.Context, text string, allowNew bool) (id int64, err error)
	AcceptNgrams(ctx context.Context, texts []string, allowNew bool) (ids []int64, err error)
	// Encode ngram
	EncodeNgram(ctx context.Context, text string) (id int64, err error)
	// Decode ngram
	DecodeNgram(ctx context.Context, id int64) (text string, err error)
	// Clear all data
	Purge() error
}

type DrugRepository interface {
	// Find existing drug scope or create new
	Scope(id int) (DrugScope, error)

	// Replace document
	Replace(ctx context.Context, doc *Doc, body string) (int64, error)
	// Fetch list of documents
	Fetch(ctx context.Context, list Results, tuner Tuner) (Results, error)
	// Update document
	Update(ctx context.Context, id int64, body string) error

	// Find document by internal code
	Find(id int64) (*Doc, error)
	// Find document by external code
	FindEx(id string) (*Doc, error)
	// Find documents by external codes
	FindExAll(drugs []string) ([]*Doc, error)
	// Find identifier by drug code
	FindByDrug(drug string) ([]int64, error)
	// Find identifier by bar code
	FindByBarCode(barCode string) ([]int64, error)
	// Find identifier by parcel identifier
	FindByParcelCode(parcel string) (int64, error)
	// Find identifier by parcel identifier (external)
	FindByParcelCodeEx(parcel string) ([]int64, error)
	// Encode external identifier
	Encode(id string) (res int64, err error)
	// Decode internal identifier
	Decode(id int64) (res string, err error)
	// Append document
	Append(doc *Doc) (id int64, err error)
	// Remove document
	Remove(id int64) (id *Doc, err error)
	// Remove all documents and scopes
	Purge() error
}

*/
