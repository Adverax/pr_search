package parcels

import (
	"context"
	"fmt"
	"math"
	"strings"

	"spWebFront/FrontKeeper/infrastructure/log"
	"spWebFront/FrontKeeper/server/app/domain/model"
	"spWebFront/FrontKeeper/server/app/domain/repository"
)

// Strategy is implementation of Strategy interface.
type strategy struct {
	Rule    Rule    // Root rule
	Mutator Mutator // Middleware func
	reader  Reader
}

func (strategy *strategy) Purge(
	ctx context.Context,
) error {
	return strategy.Rule.Purge(ctx)
}

func (strategy *strategy) Append(
	ctx context.Context,
	doc *Doc,
) {
	name := strategy.reader(doc)
	runes := strategy.prepare(ctx, name)
	strategy.Rule.Append(ctx, doc.Id, runes, 1)
}

func (strategy *strategy) Remove(
	ctx context.Context,
	id int64,
) {
	strategy.Rule.Remove(ctx, id)
}

func (strategy *strategy) Search(
	ctx context.Context,
	manager Manager,
	query string,
	details *Details,
) (Hypotheses, error) {
	runes := strategy.prepare(ctx, query)
	resolver, err := newResolver(ctx, manager, details.Filter)
	if err != nil {
		return nil, fmt.Errorf("newResolver: %w", err)
	}
	defer resolver.Close(ctx)
	return resolver.Resolve(ctx, strategy.Rule, runes, 1, details), nil
}

func (strategy *strategy) Log(ctx context.Context) {
	strategy.Rule.Log()
}

func (strategy *strategy) prepare(ctx context.Context, s string) []rune {
	return strategy.Mutator.Mute(ctx, []rune(s))
}

/*
func (strategy *strategy) Load(r io.Reader) error {
	dec := gob.NewDecoder(r)
	err := dec.Decode(strategy)
	if err != nil {
		return fmt.Errorf("Strategy.Load: %w", err)
	}
	return nil
}

func (strategy *strategy) Save(w io.Writer) error {
	enc := gob.NewEncoder(w)
	err := enc.Encode(strategy)
	if err != nil {
		return fmt.Errorf("Save: %w", err)
	}
	return nil
}*/

// NewStrategy is constructor for creating instance of strategy.
func NewStrategy(
	mutator Mutator,
	rule Rule,
	reader Reader,
) Strategy {
	if mutator == nil {
		mutator = defaultMutator
	}

	return &strategy{
		Rule:    rule,
		Mutator: mutator,
		reader:  reader,
	}
}

// NewStrategySimple is constructor for creating instance of simple strategy.
func NewStrategySimple(
	docs DocManager,
	reader Reader,
) Strategy {
	return NewStrategy(
		nil,
		NewNgramRule(
			"ngram.3",
			NewNgramIndex(docs, NgramIndexPositions{}),
			NewNgramParserPrimary(3, NewParserEstimatorPrimary(0.5)),
		),
		reader,
	)
}

type muteRu struct {
}

func (m *muteRu) Mute(ctx context.Context, runes []rune) []rune {
	return MetaphoneRu(runes)
}

type muteUa struct {
}

func (m *muteUa) Mute(ctx context.Context, runes []rune) []rune {
	return MetaphoneUa(runes)
}

var ruMute = new(muteRu)
var uaMute = new(muteUa)

type predicateRu struct {
}

func (p *predicateRu) Test(ctx context.Context, runes []rune) bool {
	return dictRu.IsValid(runes)
	//return ((ctx.language == "") || (ctx.language == Rus)) && dictRu.IsValid(runes)
}

type predicateUa struct {
}

func (p *predicateUa) Test(ctx context.Context, runes []rune) bool {
	return dictUa.IsValid(runes)
	//	return ((ctx.language == "") || (ctx.language == Ukr)) && dictUa.IsValid(runes)
}

var ruPredicate = new(predicateRu)
var uaPredicate = new(predicateUa)

type MetaphoneOptions struct {
	Original  float64 `json:"original"`  // Вес оригинальной ветки
	Russian   float64 `json:"russian"`   // Вес ветки русского метафона
	Ukrainian float64 `json:"ukrainian"` // Вес ветки украинского метафона
}

type NgramPositionBranchOptions struct {
	Weight    float64 `json:"weight"`    // Весовой коэффициент позиционной информации [0..1]
	Query     float64 `json:"query"`     // Весовой коеффициент запроса в дополнении к весовому коеффициенту образца [0,,1]
	Absolute  float64 `json:"absolute"`  // Весовой коеффициент абсолютной позиции
	Relative  float64 `json:"relative"`  // Весовой коэффициент относительной позиции
	Inflation float64 `json:"inflation"` // Скорость инфляции оценки от позиции [0..1]
}

type NgramBranchOptions struct {
	Min      int                        `json:"min"`      // Минимальная длина ngram [2..10]
	Max      int                        `json:"max"`      // Максимальная длина ngram [2..10]
	Grow     float64                    `json:"grow"`     // Шаг приращения веса более длинной ngram [1..]
	Weight   float64                    `json:"weight"`   // Вес ветки
	Position NgramPositionBranchOptions `json:"position"` // Позиционная информация
}

func (options *NgramBranchOptions) newNgrams(
	docs DocManager,
	name string,
	weight float64,
	newParser func(len int, home float64) NgramParser,
) []*Entry {
	if weight <= 0 {
		return nil
	}

	home := options.Position.Absolute + options.Position.Relative
	if home != 0 {
		home = options.Position.Absolute / home
	}

	var entries []*Entry
	length := options.Max - options.Min
	ww := weight / float64(math.Pow(float64(options.Grow), float64(length)))
	log.Debugf("NEW NGRAMS FAMILY %s, weight=%g", name, weight)
	for i := options.Min; i <= options.Max; i++ {
		if ww > 0 {
			log.Debugf("NEW NGRAMS RULE %s%d, weight=%g ", name, i, ww)
			entries = append(
				entries,
				&Entry{
					Weight: ww,
					Rule: NewNgramRule(
						fmt.Sprintf("%s%d", name, i),
						NewNgramIndex(
							docs,
							NgramIndexPositions{
								Weight:    options.Position.Weight,
								Query:     options.Position.Query,
								Inflation: options.Position.Inflation,
							},
						),
						newParser(i, home),
					),
				},
			)
		}
		ww *= options.Grow
	}

	return entries
}

type NgramOptions struct {
	Disabled  bool               `json:"disabled"`  // Ngram searc is disabled
	Mix       bool               `json:"mix"`       // Разрешить режим микширования
	Primary   NgramBranchOptions `json:"primary"`   // Ведущая ветка
	Secondary NgramBranchOptions `json:"secondary"` // Ведомая ветка
}

func (options *NgramOptions) newNgrams(
	docs DocManager,
	name string,
) Entries {
	weight := options.Primary.Weight + options.Secondary.Weight
	entries1 := options.Primary.newNgrams(
		docs,
		name+".p",
		options.Primary.Weight/weight,
		func(length int, home float64) NgramParser {
			return NewNgramParserPrimary(
				length,
				NewParserEstimatorPrimary(
					home,
				),
			)
		},
	)
	entries2 := options.Secondary.newNgrams(
		docs,
		name+".s",
		options.Secondary.Weight/weight,
		func(length int, home float64) NgramParser {
			return NewNgramParserSecondary(
				length,
				NewParserEstimatorSecondary(),
			)
		},
	)
	entries := make(Entries)
	for _, e := range entries1 {
		entries[e.Name()] = e
	}
	for _, e := range entries2 {
		entries[e.Name()] = e
	}
	return entries
}

type NgramTranslators struct {
	Weight   float64              `json:"weight"`
	Keyboard NgramKeyboardOptions `json:"keyboard"`
	Phonetic NgramPhoneticOptions `json:"phonetic"`
}

type NgramKeyboardOptions struct {
	En2Ru bool `json:"en-ru"`
	En2Ua bool `json:"en-ua"`
	Ru2Ua bool `json:"ru-ua"`
	Ua2Ru bool `json:"ua-ru"`
}

type NgramPhoneticOptions struct {
	Ru2Ua bool `json:"ru-ua"`
	Ua2Ru bool `json:"ua-ru"`
}

type BandOptions struct {
	Capacity  int             `json:"capacity"`  // Maximal count in band. Default 100
	Threshold float64         `json:"threshold"` // The absolute relevance value for accept
	Diff      BandDiffOptions `json:"diff"`      // Differential
}

type BandDiffOptions struct {
	Rel float64 `json:"rel"` // The maximum difference between two adjacent lines [0..1]. Default 0.
	Abs float64 `json:"abs"` // The maximum difference between first and current lines (cur/max) [0..1]. Default 0.
}

type StrategyOptions struct {
	Ngrams      NgramOptions     `json:"ngrams"`
	Translators NgramTranslators `json:"translators"`
	Metaphone   MetaphoneOptions `json:"metaphone"`
	Band        BandOptions      `json:"band"`
	Group       bool             `json:"group"`
}

func DefaultStrategyOptions() *StrategyOptions {
	return &StrategyOptions{
		Group: true,
		Metaphone: MetaphoneOptions{
			Original:  8,
			Russian:   1,
			Ukrainian: 1,
		},
		Ngrams: NgramOptions{
			Primary: NgramBranchOptions{
				Min:    3,
				Max:    6,
				Grow:   2,
				Weight: 0.7,
				Position: NgramPositionBranchOptions{
					Weight:    0.3,
					Inflation: 0.3,
					Absolute:  1,
					Relative:  0,
				},
			},
			Secondary: NgramBranchOptions{
				Min:    3,
				Max:    6,
				Grow:   2,
				Weight: 0.3,
				Position: NgramPositionBranchOptions{
					Weight:    0.3,
					Inflation: 0.3,
					Absolute:  1,
					Relative:  0,
				},
			},
		},
		Translators: NgramTranslators{
			Weight: 0,
		},
		Band: BandOptions{
			Capacity: 100,
		},
	}
}

type muteRoot struct {
}

func (m *muteRoot) Mute(ctx context.Context, runes []rune) []rune {
	return []rune(strings.ToLower(string(runes)))
}

var rootMute = new(muteRoot)

func NewStrategyDefault(
	docs DocManager,
	options *StrategyOptions,
	reader Reader,
) Strategy {
	if options == nil {
		options = DefaultStrategyOptions()
	}

	var merge func(name string, entries Entries) Rule

	var estimator Estimator
	estimator = NewMaxEstimator()

	if options.Ngrams.Mix {
		merge = func(name string, entries Entries) Rule {
			return NewMultiRule(name+".mix", entries, estimator, NewSumMixer())
		}
	} else {
		merge = func(name string, entries Entries) Rule {
			return NewMultiRule(name+".max", entries, estimator, NewMaxMixer())
		}
	}

	var entries []*Entry
	if options.Metaphone.Original > 0 {
		entries = append(
			entries,
			&Entry{
				Weight: options.Metaphone.Original,
				// Rule: NewDistortRule(
				// 	"main.distort",
				// 	nil,
				// 	nil,
				// NewFilterRule(
				// 	"main.filter",
				Rule: merge(
					"main",
					options.Ngrams.newNgrams(docs, "main.ngram"),
				),
				//					),
				// ),
			},
		)
	}

	if options.Metaphone.Russian > 0 {
		entries = append(
			entries,
			&Entry{
				Weight: options.Metaphone.Russian,
				Rule: NewGuardRule(
					"ru.metaphone.guard",
					ruPredicate,
					ruPredicate,
					NewMuteRule(
						"ru.metaphone.distort",
						ruMute,
						ruMute,
						// NewFilterRule(
						// 	"ru.filter",
						merge(
							"ru.metaphone",
							options.Ngrams.newNgrams(docs, "ru.metaphone.ngram"),
						),
						// ),
					),
				),
			},
		)
	}

	if options.Metaphone.Ukrainian > 0 {
		entries = append(
			entries,
			&Entry{
				Weight: options.Metaphone.Ukrainian,
				Rule: NewGuardRule(
					"ua.metaphone.guard",
					uaPredicate,
					uaPredicate,
					NewMuteRule(
						"ua.metaphone.distort",
						uaMute,
						uaMute,
						// NewFilterRule(
						// 	"ua.filter",
						merge(
							"ua.metaphone",
							options.Ngrams.newNgrams(docs, "ua.metaphone.ngram"),
							//),
						),
					),
				),
			},
		)
	}

	var rule Rule
	if len(entries) == 1 {
		rule = entries[0].Rule
	} else {
		rule = NewMultiRule(
			"layout.max",
			NewEntries(entries...),
			estimator,
			NewMaxMixer(),
		)
	}

	var translators []LayoutTranslator

	if options.Translators.Keyboard.En2Ru {
		translators = append(
			translators,
			NewLayoutTranslator(layoutEn2RuKeyboard, dictEn, Rus),
		)
	}

	if options.Translators.Keyboard.En2Ua {
		translators = append(
			translators,
			NewLayoutTranslator(layoutEn2UaKeyboard, dictEn, Ukr),
		)
	}

	if options.Translators.Keyboard.Ru2Ua {
		translators = append(
			translators,
			NewLayoutTranslator(layoutRu2UaKeyboard, dictRu, Ukr),
		)
	}

	if options.Translators.Keyboard.Ua2Ru {
		translators = append(
			translators,
			NewLayoutTranslator(layoutUa2RuKeyboard, dictUa, Rus),
		)
	}

	if options.Translators.Phonetic.Ru2Ua {
		translators = append(
			translators,
			NewLayoutTranslator(layoutUa2RuPhonetic, dictRu, Ukr),
		)
	}

	if options.Translators.Phonetic.Ua2Ru {
		translators = append(
			translators,
			NewLayoutTranslator(layoutRu2UaPhonetic, dictUa, Rus),
		)
	}

	if len(translators) != 0 && options.Translators.Weight > 0 {
		rule = NewLayoutRule(
			"layout",
			rule,
			translators,
			options.Translators.Weight,
			estimator,
		)
	}

	return NewStrategy(rootMute, rule, reader)
}

/*
func init() {
	gob.Register(&strategy{})
	gob.Register(ruPredicate)
	gob.Register(uaPredicate)
	gob.Register(ruMute)
	gob.Register(uaMute)
	gob.Register(rootMute)
	gob.Register(defaultMutator)
	gob.Register(defaultPredicate)
}*/

type exactStrategy struct {
	Layouts      []LayoutTranslator
	LayoutWeight float64
	Estimator    Estimator
	docs         DocManagerEx
	reader       Reader
}

func (strategy *exactStrategy) Search(
	ctx context.Context,
	manager Manager,
	query string,
	details *Details,
) (Hypotheses, error) {
	query = strings.ToLower(query)
	hs := make([]Hypotheses, 0, len(strategy.Layouts)+1)

	// Выполняем поиск без преобразования символов
	h, err := strategy.search(ctx, manager, query, 1, details)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	if h != nil {
		hs = append(hs, h)
	}

	// Выполняем поиск для каждой из схем преобразования символов
	query2 := []rune(query)
	for _, layout := range strategy.Layouts {
		q := layout.Translate(ctx, query2, details)
		if q == nil {
			continue
		}
		w := calcWeight(string(query), string(q))
		h, err := strategy.search(ctx, manager, string(q), w, details)
		if err != nil {
			return nil, fmt.Errorf("search: %w", err)
		}
		if h != nil {
			h = h.scale(strategy.LayoutWeight)
			hs = append(hs, h)
		}
	}

	// Выполняем поиск лучшего набора гипотез
	br := float64(0)
	var bh Hypotheses
	for _, h := range hs {
		r := strategy.Estimator.Estimate(h)
		if br < r {
			br = r
			bh = h
		}
	}

	return bh, nil
}

func (strategy *exactStrategy) search(
	ctx context.Context,
	manager Manager,
	query string,
	weight float64,
	details *Details,
) (Hypotheses, error) {
	return strategy.exactSearch(
		ctx,
		func(doc *Doc) float64 {
			index := strings.ToLower(strategy.reader(doc))

			if strings.HasPrefix(index, query) {
				return 1 * weight
			}

			if strings.Contains(index, query) {
				return 0.5 * weight
			}

			return -1
		},
	)
}

func (strategy *exactStrategy) exactSearch(
	ctx context.Context,
	estimate func(doc *Doc) float64,
) (Hypotheses, error) {
	hs := newHypotheses()
	err := strategy.docs.ForEach(
		ctx,
		func(ctx context.Context, doc *Doc) error {
			relevance := estimate(doc)
			if relevance <= 0 {
				return nil
			}
			hs[doc.Id] = relevance
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("ForEach: %w", err)
	}

	return hs, nil
}

func (strategy *exactStrategy) Log(ctx context.Context) {
	// nothing
}

/*
func (strategy *exactStrategy) Load(r io.Reader) error {
	return nil
}

func (strategy *exactStrategy) Save(w io.Writer) error {
	return nil
}*/

func (strategy *exactStrategy) Purge(ctx context.Context) error {
	return nil
}

func (strategy *exactStrategy) Append(
	ctx context.Context,
	doc *Doc,
) {
	// nothing
}

func (strategy *exactStrategy) Remove(
	ctx context.Context,
	id int64,
) {
	// nothing
}

func NewExactStrategy(
	docs DocManagerEx,
	options *StrategyOptions,
	reader Reader,
) Strategy {
	var translators []LayoutTranslator

	if options.Translators.Keyboard.En2Ru {
		translators = append(
			translators,
			NewLayoutTranslator(layoutEn2RuKeyboard, dictEn, Rus),
		)
	}

	if options.Translators.Keyboard.En2Ua {
		translators = append(
			translators,
			NewLayoutTranslator(layoutEn2UaKeyboard, dictEn, Ukr),
		)
	}

	if options.Translators.Keyboard.Ru2Ua {
		translators = append(
			translators,
			NewLayoutTranslator(layoutRu2UaKeyboard, dictRu, Ukr),
		)
	}

	if options.Translators.Keyboard.Ua2Ru {
		translators = append(
			translators,
			NewLayoutTranslator(layoutUa2RuKeyboard, dictUa, Rus),
		)
	}

	if options.Translators.Phonetic.Ru2Ua {
		translators = append(
			translators,
			NewLayoutTranslator(layoutUa2RuPhonetic, dictRu, Ukr),
		)
	}

	if options.Translators.Phonetic.Ua2Ru {
		translators = append(
			translators,
			NewLayoutTranslator(layoutRu2UaPhonetic, dictUa, Rus),
		)
	}

	return &exactStrategy{
		Layouts:      translators,
		LayoutWeight: options.Translators.Weight,
		Estimator:    NewMaxEstimator(),
		reader:       reader,
		docs:         docs,
	}
}

type multiStrategy []Strategy

func (strategy multiStrategy) Log(
	ctx context.Context,
) {
	for _, s := range strategy {
		s.Log(ctx)
	}
}

func (strategy multiStrategy) Purge(
	ctx context.Context,
) error {
	for _, s := range strategy {
		err := s.Purge(ctx)
		if err != nil {
			return fmt.Errorf("Purge: %w", err)
		}
	}
	return nil
}

func (strategy multiStrategy) Append(
	ctx context.Context,
	doc *Doc,
) {
	for _, s := range strategy {
		s.Append(ctx, doc)
	}
}

func (strategy multiStrategy) Remove(
	ctx context.Context,
	id int64,
) {
	for _, s := range strategy {
		s.Remove(ctx, id)
	}
}

func (strategy multiStrategy) Search(
	ctx context.Context,
	manager Manager,
	query string,
	details *Details,
) (Hypotheses, error) {
	hs := make(Hypotheses)
	for _, s := range strategy {
		hss, err := s.Search(ctx, manager, query, details)
		if err != nil {
			return nil, fmt.Errorf("Search: %w", err)
		}
		hs.extends(hss)
	}
	return hs, nil
}

type StrategyBuilder func(options *StrategyOptions, reader Reader) Strategy

func NewMultiStrategy(
	ss ...Strategy,
) Strategy {
	if len(ss) == 1 {
		return ss[0]
	}

	res := make(multiStrategy, 0, len(ss))
	for _, s := range ss {
		res = append(res, s)
	}

	return res
}

func NewMultiStrategyWithReaders(
	builder StrategyBuilder,
	options *StrategyOptions,
	readers []Reader,
) Strategy {
	if len(readers) == 1 {
		return builder(options, readers[0])
	}

	ss := make(multiStrategy, 0)
	for _, reader := range readers {
		s := builder(options, reader)
		ss = append(ss, s)
	}

	return ss
}

func GetNameSearchIndexMultiLangReaders() []Reader {
	return []Reader{
		docNameSearchIndexReader,
		docNameSearchIndexReader2,
		docNameSearchIndexReader3,
	}
}

func GetInnSearchIndexMultiLangReaders() []Reader {
	return []Reader{
		docInnSearchIndexReader,
		docInnSearchIndexReader2,
		docInnSearchIndexReader3,
	}
}

type innStrategy struct {
	reader Reader
	docs   DocManagerEx
}

func (strategy *innStrategy) Log(
	ctx context.Context,
) {
}

func (strategy *innStrategy) Purge(
	ctx context.Context,
) error {
	return nil
}

func (strategy *innStrategy) Append(
	ctx context.Context,
	doc *Doc,
) {
}

func (strategy *innStrategy) Remove(
	ctx context.Context,
	id int64,
) {
}

func (strategy *innStrategy) Search(
	ctx context.Context,
	manager Manager,
	query string,
	details *Details,
) (Hypotheses, error) {
	hs := make(Hypotheses, details.Band.Capacity)

	err := strategy.docs.ForEach(
		ctx,
		func(ctx context.Context, doc *Doc) error {
			s := strategy.reader(doc)
			if s == query {
				hs[doc.Id] = 1
			}
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("ForEach: %w", err)
	}

	return hs, nil
}

func NewInnStrategy(
	docs DocManagerEx,
	options *StrategyOptions,
	reader Reader,
) Strategy {
	return &innStrategy{
		reader: reader,
		docs:   docs,
	}
}

type makerStrategy struct {
	parcels repository.ParcelRepository
}

func (strategy *makerStrategy) Log(
	ctx context.Context,
) {
}

func (strategy *makerStrategy) Purge(
	ctx context.Context,
) error {
	return nil
}

func (strategy *makerStrategy) Append(
	ctx context.Context,
	doc *Doc,
) {
}

func (strategy *makerStrategy) Remove(
	ctx context.Context,
	id int64,
) {
}

func (strategy *makerStrategy) Search(
	ctx context.Context,
	manager Manager,
	query string,
	details *Details,
) (Hypotheses, error) {
	ids, err := strategy.parcels.FindByMaker(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("FindByMaker: %d", err)
	}
	return makeHypotheses(ids, 0.5), nil
}

func NewMakerStrategy(
	parcels repository.ParcelRepository,
	options *StrategyOptions,
) Strategy {
	return &makerStrategy{
		parcels: parcels,
	}
}

// Данная стратегия позволяет искать данные непосредственно в базе данных по совпадению с началом/вхождению.
type primarySearchStrategy struct {
	parcels repository.ParcelRepository
	options PrimarySearchOptions
}

func (strategy *primarySearchStrategy) Log(
	ctx context.Context,
) {
}

func (strategy *primarySearchStrategy) Purge(
	ctx context.Context,
) error {
	return nil
}

func (strategy *primarySearchStrategy) Append(
	ctx context.Context,
	doc *Doc,
) {
}

func (strategy *primarySearchStrategy) Remove(
	ctx context.Context,
	id int64,
) {
}

func (strategy *primarySearchStrategy) Search(
	ctx context.Context,
	manager Manager,
	query string,
	details *Details,
) (Hypotheses, error) {

	// if details.Lang < 0 || details.Lang >= len(strategy.options.Langs) {
	// 	return nil, fmt.Errorf("Invalid target language %d", details.Lang)
	// }

	return strategy.parcels.FindByPattern(
		ctx,
		query,
		"search",
		model.ParcelQueryOptions{
			Starts:   strategy.options.Starts,
			Contains: strategy.options.Contains,
			Fields:   strategy.options.Fields, //strategy.options.Langs[details.Lang],
			Exact:    strategy.options.Exact,
		},
	)
}

type PrimarySearchOptions struct {
	Fields   []string
	Starts   float32
	Contains float32
	Exact    float32
	//Langs    []string
}

func NewPrimarySearchStrategy(
	parcels repository.ParcelRepository,
	options PrimarySearchOptions,
) Strategy {
	return &primarySearchStrategy{
		parcels: parcels,
		options: options,
	}
}

type searchers []Searcher

func (searchers searchers) Search(
	ctx context.Context,
	manager Manager,
	query string,
	details *Details,
) (Hypotheses, error) {
	hs := make(Hypotheses)
	for _, s := range searchers {
		hss, err := s.Search(ctx, manager, query, details)
		if err != nil {
			return nil, fmt.Errorf("Search: %w", err)
		}
		hs.extends(hss)
	}
	return hs, nil
}

type SearcherBuilder func(reader Reader) Searcher

func NewSearchers(
	builder SearcherBuilder,
	readers []Reader,
) Searcher {
	if len(readers) == 1 {
		return builder(readers[0])
	}

	multi := make(searchers, 0)
	for _, reader := range readers {
		s := builder(reader)
		multi = append(multi, s)
	}
	return multi
}
