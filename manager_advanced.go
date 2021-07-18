package parcels

import (
	"context"
	"encoding/gob"
	"fmt"
	"sort"
	"spWebFront/FrontKeeper/infrastructure/core"
	"spWebFront/FrontKeeper/infrastructure/log"
	"spWebFront/FrontKeeper/infrastructure/workflow/thread"
	"spWebFront/FrontKeeper/server/app/domain/model"
	"spWebFront/FrontKeeper/server/app/domain/repository"
	"spWebFront/FrontKeeper/server/app/domain/service/storage"
	"strings"
	"sync"
	"time"
)

// Field reader
type Reader func(doc *Doc) string

// Strategy for abstract search
type Strategy interface {
	Searcher
	// Log info about usage
	Log(ctx context.Context)
	// Purge index
	Purge(ctx context.Context) error
	// Replace document (or append)
	Append(ctx context.Context, doc *Doc)
	// Remove existing document
	Remove(ctx context.Context, id int64)
	// Load data from reader
	//Load(r io.Reader) error
	// Save data to writer
	//Save(w io.Writer) error
}

// Rule is single rule of search strategy.
type Rule interface {
	Clone() Rule
	// Get rule name
	Name() string
	// Purge all data
	Purge(ctx context.Context) error
	// Append new document
	Append(ctx context.Context, id int64, name []rune, weight float64)
	// Remove old document
	Remove(ctx context.Context, id int64)
	// Make hypotheses
	Search(ctx context.Context, resolver Resolver, query []rune, weight float64, details *Details) Hypotheses
	// Log info
	Log()
}

// Entry for single rule
type Entry struct {
	Rule           // Rule
	Weight float64 // Weight of the rule
}

// Entries is map of rule entry
type Entries map[string]*Entry // name -> entry

func (entries Entries) Clone() Entries {
	res := make(Entries, len(entries))
	for k, e := range entries {
		res[k] = &Entry{
			Rule:   e.Rule,
			Weight: e.Weight,
		}
	}
	return res
}

// NewEntries is constructor for create map of entries.
func NewEntries(entries ...*Entry) Entries {
	res := make(Entries, len(entries))

	isList := true
	for _, e := range entries {
		if e.Weight != 0 {
			isList = false
			break
		}
	}

	n := float64(len(entries))
	if isList {
		for _, e := range entries {
			res[e.Name()] = e
			e.Weight = 1 / n
		}
	} else {
		var s float64
		for _, e := range entries {
			s += e.Weight
		}

		for _, e := range entries {
			e.Weight /= s
			res[e.Name()] = e
		}
	}

	return res
}

// Mutator is abstract translator rules.
type Mutator interface {
	Mute(ctx context.Context, rs []rune) []rune
}

// Predicate is abstract guard for runes.
type Predicate interface {
	Test(ctx context.Context, rs []rune) bool
}

type Strategies struct {
	sync.RWMutex
	docs   DocManager
	Names  Strategy
	Inns   Strategy
	Makers Strategy
}

func (strategies *Strategies) Purge(ctx context.Context) error {
	err := strategies.Names.Purge(ctx)
	if err != nil {
		return fmt.Errorf("Purge: %w", err)
	}
	return strategies.docs.Purge(ctx)
}

func (strategies *Strategies) Log(ctx context.Context) error {
	strategies.Names.Log(ctx)
	return nil
}

func (strategies *Strategies) Append(ctx context.Context, doc *Doc) error {
	err := strategies.docs.Append(ctx, doc)
	if err != nil {
		return fmt.Errorf("Append: %w", err)
	}
	strategies.Names.Append(ctx, doc)
	return nil
}

func (strategies *Strategies) Remove(ctx context.Context, id int64) error {
	strategies.Names.Remove(ctx, id)
	return strategies.docs.Remove(ctx, id)
}

func (strategies *Strategies) getNames() Strategy {
	strategies.RLock()
	p := strategies.Names
	strategies.RUnlock()
	return p
}

// Engine is thread safe implementation of Manager.
type advancedEngine struct {
	baseEngine
	strategies *Strategies
	docs       DocManager
}

func (engine *advancedEngine) Search(
	ctx context.Context,
	query string,
	typ string,
	details *Details,
) (res model.Parcels, err error) {
	if details == nil {
		details = new(Details)
		engine.InitDetails(details)
	}

	if details.Exact == 0 {
		details.Exact = 0.5
	}

	if details.Band.Capacity == 0 {
		details.Band.Capacity = 100
	}

	if details.Weights == nil {
		details.Weights = defaultWeights
	}

	engine.RLock()
	defer engine.RUnlock()

	query = strings.TrimSpace(strings.ToLower(query))
	hs := make(Hypotheses)
	paradigm := getParadigm(typ)
	if len(paradigm.types) == 0 {
		hs, err = engine.search(ctx, paradigm.method, query, details)
		if err != nil {
			return nil, fmt.Errorf("search (%s): %w", typ, err)
		}
	} else {
		for _, typ := range paradigm.types {
			p := getParadigm(typ)

			hss, err := engine.search(ctx, p.method, query, details)
			if err != nil {
				return nil, fmt.Errorf("search (%s): %w", typ, err)
			}

			hs.extends(hss)

			if details.IsCancel() {
				return nil, nil
			}
		}
	}

	ps, err := engine.docs.Resolve(ctx, hs, details, paradigm.group)
	if err != nil {
		return nil, fmt.Errorf("Resolve: %w", err)
	}

	if debug {
		log.Debugf("SEARCH result %d/%d ", len(ps), len(hs))
	}
	return ps, nil
}

func (engine *advancedEngine) search(
	ctx context.Context,
	method method,
	query string,
	details *Details,
) (
	hs Hypotheses,
	err error,
) {
	//	starts := time.Now()
	engine.searchEnter()
	defer func() {
		engine.searchLeave()
		// finished := time.Now().Sub(starts)
		// log.Debugln("SEARCH TIME ", finished)
	}()

	hs, err = method(ctx, engine, query, details)
	if err != nil {
		return nil, err
	}

	if hs == nil {
		hs = newHypotheses()
	}

	return
}

/*
func (engine *engine) Load(ctx context.Context) error {
	engine.Lock()
	defer engine.Unlock()

	err := engine.load(ctx)
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}

	return nil
}

func (engine *engine) Save(ctx context.Context) error {
	engine.Lock()
	defer engine.Unlock()

	err := engine.save(ctx)
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}
*/

/*
func (engine *engine) save(
	ctx context.Context,
) error {
	// Save documents
	{
		var documents bytes.Buffer
		if err := engine.docs.Save(&documents); err != nil {
			return fmt.Errorf("save documents: %w", err)
		}
		if err := engine.binary.Put(ctx, namespace, keyDocuments, documents.Bytes()); err != nil {
			return fmt.Errorf("Put documents: %w", err)
		}
	}

	// Save name index
	{
		var names bytes.Buffer
		if err := engine.names.Save(&names); err != nil {
			return fmt.Errorf("save names: %w", err)
		}
		if err := engine.binary.Put(ctx, namespace, keyNameIndex, names.Bytes()); err != nil {
			return fmt.Errorf("Put names", err)
		}
	}

	return nil
}

func (engine *engine) load(
	ctx context.Context,
) error {
	var names Strategy

	dataDocuments, err := engine.binary.Get(ctx, namespace, keyDocuments)
	if err != nil {
		return fmt.Errorf("Get documents: %w", err)
	}
	if dataDocuments == nil {
		return nil
	}

	dataNameIndex, err := engine.binary.Get(ctx, namespace, keyNameIndex)
	if err != nil {
		return fmt.Errorf("Get names: %w", err)
	}
	if dataNameIndex == nil {
		return nil
	}

	// Load name index
	names, err = engine.factory.New()
	if err != nil {
		return fmt.Errorf("create names: %w", err)
	}

	err = names.Load(bytes.NewBuffer(dataNameIndex))
	if err != nil {
		return fmt.Errorf("load name index: %w", err)
	}

	// Load documents
	err = engine.docs.Load(bytes.NewBuffer(dataDocuments))
	if err != nil {
		return fmt.Errorf("decode documents: %w", err)
	}

	engine.names = names

	return nil
}

func (engine *engine) Migrate(
	ctx context.Context,
	factory StrategyFactory,
) error {
	engine.Lock()
	defer engine.Unlock()

	if engine.factory == factory {
		return nil
	}
	// ctx, err := engine.newContext()
	// if err != nil {
	// 	return err
	// }

	return engine.db.Update(
		func(tx bolt.Tx) error {
			names, err := factory.New()
			if err != nil {
				return fmt.Errorf("create names", err)
			}

			access := engine.documents.Capture()
			defer access.Release()

			src := access.Items()
			dst := document.New()
			for _, doc := range src {
				id := dst.Append(doc)
				names.Append(ctx, id, doc.NameSearchIndex)
			}

			b, err := tx.CreateBucketIfNotExists(bucketNameBinary)
			if err != nil {
				return fmt.Errorf("create bucket %q: %w", string(bucketNameBinary), err)
			}

			// Update name index
			err = b.Delete(keyNameIndex)
			if err != nil {
				return fmt.Errorf("remove name index: %w", err)
			}

			var dataName bytes.Buffer
			err = names.Save(&dataName)
			if err != nil {
				return fmt.Errorf("save names: %w", err)
			}

			err = b.Put(keyNameIndex, dataName.Bytes())
			if err != nil {
				return fmt.Errorf("put name index: %w", err)
			}

			engine.documents = dst
			engine.names = names

			return nil
		},
	)
}*/

func (engine *advancedEngine) FindByParcelCode(
	ctx context.Context,
	parcel string,
) ([]int64, error) {
	id, err := engine.parcels.FindByParcelCode(ctx, parcel)
	if err != nil {
		return nil, fmt.Errorf("FindByParcelCode: %w", err)
	}

	if id == 0 {
		return make([]int64, 0), nil
	}

	return []int64{id}, nil
}

// NewManager is constructor for creating instance of manager.
func NewAdvancedManager(
	ctx context.Context,
	auths core.AuthManager,
	parcels repository.ParcelRepository,
	storage storage.Manager,
	messenger Messenger,
	notifier core.Notifier,
	docs DocManager,
	strategies *Strategies,
	threads thread.Activator,
	options Options,
) (Manager, error) {
	lang, err := storage.GetString(ctx, namespace, keyLanguage, Rus)
	if err != nil {
		return nil, fmt.Errorf("GetString: %w", err)
	}

	options.Stocks.Lang = lang

	sReady := make(chan bool, 64)
	sReady <- true

	e := &advancedEngine{
		baseEngine: baseEngine{
			control: control{
				ready: sReady,
			},
			behavior:  strategies,
			auths:     auths,
			storage:   storage,
			parcels:   parcels,
			messenger: messenger,
			notifier:  notifier,
			options:   options,
			disabled:  1,
		},
		strategies: strategies,
		docs:       docs,
	}

	// initialize engine
	raw, err := parcels.FindAllRaw(ctx)
	if err != nil {
		return nil, fmt.Errorf("Find parcels: %w", err)
	}

	defs, err := makeDocDefs(raw)
	if err != nil {
		return nil, fmt.Errorf("makeDocDefs: %w", err)
	}

	err = e.Refresh(ctx, defs)
	if err != nil {
		return nil, fmt.Errorf("Load parcels: %w", err)
	}

	e.initialized = true

	if threads == nil {
		return e, nil
	}

	threads.Periodic(
		ctx,
		thread.Identity{Name: ManagerEntityId},
		e.step,
		time.Duration(options.Stocks.Interval)*time.Second,
		core.NewErrorHandler(ctx, core.ErrorStocks, "parcels.updater"),
	)

	return e, nil
}

func makeDocDefs(parcels []*model.Raw) (res []*DocDef, err error) {
	res = make([]*DocDef, 0, len(parcels))
	for _, p := range parcels {
		var doc Doc
		err := core.JsonUnmarshal([]byte(p.Document), &doc)
		if err != nil {
			return nil, fmt.Errorf("Unmarshal: %w", err)
		}
		doc.Id = p.Id
		res = append(
			res,
			&DocDef{
				Head: doc,
				Body: p.Document,
			},
		)
	}
	return
}

type method func(
	ctx context.Context,
	engine *advancedEngine,
	query string,
	details *Details,
) (Hypotheses, error)

type paradigm struct {
	types  []string
	group  Reader
	method method
}

var paradigms = map[string]paradigm{
	"inn": {
		group:  docInnGroupIndexReader,
		method: doInnSearch,
	},
	"parcel": {
		group:  docParcelCodeExReader,
		method: doParcelCodeExSearch,
	},
	"code": {
		group:  docParcelCodeExReader,
		method: doParcelCodeExSearch,
	},
	"maker": {
		group:  docMakerReader,
		method: doMakerSearch,
	},
	"barcode": {
		group:  docBarCodeReader,
		method: doBarCodeSearch,
	},
	"mix": {
		group: docNameGroupIndexReader,
		types: []string{"name" /*"inn",*/, "maker", "barcode"},
	},
	"name": {
		group:  docNameGroupIndexReader,
		method: doMultiSearch,
	},
	"default": {
		group:  docNameGroupIndexReader,
		method: doMultiSearch,
	},
}

func getParadigm(typ string) paradigm {
	// Predefined paradigms
	if p, ok := paradigms[typ]; ok {
		return p
	}

	// Complex custom paradigm
	res := paradigm{
		group: docNameGroupIndexReader,
	}
	ss := strings.Split(typ, " ")
	for _, s := range ss {
		if _, ok := paradigms[s]; ok {
			res.types = append(res.types, s)
		}
	}
	if len(res.types) != 0 {
		return res
	}

	// Default paradigm
	return paradigms["default"]
}

func doInnSearch(
	ctx context.Context,
	engine *advancedEngine,
	query string,
	details *Details,
) (Hypotheses, error) {
	hs, err := engine.strategies.Inns.Search(ctx, engine, query, details)
	if err != nil {
		return nil, fmt.Errorf("findByInn: %w", err)
	}

	return hs, err
}

func doParcelCodeExSearch(
	ctx context.Context,
	engine *advancedEngine,
	query string,
	details *Details,
) (Hypotheses, error) {
	ids, err := engine.parcels.FindByParcelCodeEx(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("FindByParcelCodeEx: %w", err)
	}

	return makeHypotheses(ids, 1), nil
}

func doMakerSearch(
	ctx context.Context,
	engine *advancedEngine,
	query string,
	details *Details,
) (Hypotheses, error) {
	return engine.strategies.Makers.Search(ctx, engine, query, details)
}

func doBarCodeSearch(
	ctx context.Context,
	engine *advancedEngine,
	query string,
	details *Details,
) (Hypotheses, error) {
	ids, err := engine.parcels.FindByBarCode(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("FindByBarCode: %w", err)
	}

	return makeHypotheses(ids, 1), nil
}

func doNameSearch(
	ctx context.Context,
	engine *advancedEngine,
	query string,
	details *Details,
) (Hypotheses, error) {
	hs, err := engine.strategies.Names.Search(ctx, engine, query, details)
	if err != nil {
		return nil, fmt.Errorf("names.Search: %w", err)
	}

	return hs, nil
}

func doMultiSearch(
	ctx context.Context,
	engine *advancedEngine,
	query string,
	details *Details,
) (Hypotheses, error) {
	if numberRe.Match([]byte(query)) {
		ids1, err := engine.parcels.FindByBarCode(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("FindByBarCode: %w", err)
		}

		hs1 := makeHypotheses(ids1, 1)

		ids2, err := engine.parcels.FindByParcelCodeEx(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("FindByParcelCodeEx: %w", err)
		}

		hs2 := makeHypotheses(ids2, 1)

		hs1.extends(hs2)

		return hs1, nil
	}

	if model.IsGuid(query) {
		ids, err := engine.FindByParcelCode(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("FindByParcelCode: %w", err)
		}

		return makeHypotheses(ids, 1), nil
	}

	return doNameSearch(ctx, engine, query, details)
}

func makeHypotheses(
	ids []int64,
	relevance float64,
) Hypotheses {
	hs := make(Hypotheses, len(ids))
	for _, id := range ids {
		hs[id] = relevance
	}
	return hs
}

// Mixer is abstract interface for mix set of hypotheses.
type branch struct {
	hypotheses Hypotheses
	relevance  float64
}

type Mixer interface {
	Mix(branches []branch) Hypotheses
}

// Estimator is abstract interface for estimate multiple hypotheses/
type Estimator interface {
	Estimate(Hypotheses) float64
}

// Hypotheses is set of hypotheses (document and relevance).
type Hypotheses map[int64]float64

// Extends hypotheses by new hypotheses
func (hs Hypotheses) extends(hss Hypotheses) {
	for id, rel := range hss {
		if old, has := hs[id]; has && rel < old {
			continue
		}
		hs[id] = rel
	}
}

// Scale set of hypotheses
func (hs Hypotheses) scale(ratio float64) Hypotheses {
	rs := make(Hypotheses, len(hs))
	for k, v := range hs {
		rs[k] = ratio * v
	}
	return rs
}

// Build list of versions (sorted by relevance).
/*func (hs Hypotheses) versions(
	documents map[int64]*Doc,
	//tolerance float32,
	reader Reader,
) versions {
	//threshold := 1 - tolerance
	i := 0
	vs := make(versions, len(hs))
	for k, v := range hs {
		doc := documents[k]
		// if v < threshold {
		// 	continue
		// }
		group := reader(doc)
		vs[i] = &version{
			doc:       doc,
			relevance: v,
			len:       len(group),
			group:     group,
			name:      doc.NameLong,
		}
		i++
	}
	if i < len(hs) {
		vs = vs[:i]
	}
	sortVersions(vs)
	return vs
}*/

// Filter hypotheses by sensitivity.
func (hs Hypotheses) filter(sensitivity float64) Hypotheses {
	threshold := 1 - sensitivity
	res := make(Hypotheses, len(hs))
	for k, v := range hs {
		if v >= threshold {
			res[k] = v
		}
	}
	return res
}

// Promote by barcode
/*func (hs Hypotheses) promoteBarcode(
	documents document.Access,
	query string,
	score float32,
) Hypotheses {
	docs := documents.FindByBarCode(query)
	if len(docs) == 0 {
		return hs
	}

	vs := make(Hypotheses, len(hs))
	for id, h := range hs {
		if list.Contains(docs, id) {
			h += score
		}
		vs[id] = h
	}

	for _, doc := range docs {
		if _, ok := hs[doc]; !ok {
			vs[doc] = score
		}
	}

	return vs
}

// Promote by parcel identifier
func (hs Hypotheses) promoteParcel(
	documents document.Access,
	query string,
	score float32,
) Hypotheses {
	docs := documents.FindByParcelCodeEx(query)
	if len(docs) == 0 {
		return hs
	}

	vs := make(Hypotheses, len(hs))
	for id, h := range hs {
		if list.Contains(docs, id) {
			h += score
		}
		vs[id] = h
	}

	for _, doc := range docs {
		if _, ok := hs[doc]; !ok {
			vs[doc] = score
		}
	}

	return vs
}

// Promote by starts characters
func (hs Hypotheses) promoteExactStarts(
	documents document.Access,
	query string,
	extractor func(doc *document.Doc) string,
	score float32, // must be [0..1]
) Hypotheses {
	items := documents.Items()
	query = strings.ToLower(query)
	tokens := strings.Split(query, " ")

	var cnt float32
	for _, t := range tokens {
		if len(t) != 0 {
			cnt += 1.0
		}
	}
	if cnt == 0 {
		return hs
	}

	vs := hs.scale(1 - score)
	for _, t := range tokens {
		if len(t) == 0 {
			continue
		}

		for k, v := range vs {
			doc, ok := items[k]
			if !ok {
				continue
			}
			name := strings.ToLower(doc.SearchIndex)
			index := strings.Index(name, t)
			if index >= 0 {
				d := extractor(doc)
				l := len(d)
				s := score / (float32(l) * cnt)
				v1 := s * float32(l-index)
				vs[k] = v + v1
			}
		}
	}

	return vs
}*/

func init() {
	gob.Register(&Entry{})
	gob.Register(&Entries{})
	gob.Register(&maxMixer{})
	gob.Register(&sumMixer{})
	gob.Register(&maxEstimator{})
}

// Abstract rule resolver
type Resolver interface {
	Close(ctx context.Context)
	Resolve(ctx context.Context, ruley Rule, query []rune, weight float64, details *Details) Hypotheses
}

// Implementation of resolver
type resolver struct {
	// cache   map[Rule]map[string]Hypotheses
	manager Manager
	model.EntityFilterEx
	docs map[int64]*Doc
}

func (res *resolver) Close(
	ctx context.Context,
) {
	res.EntityFilterEx.Release(ctx)
}

func (res *resolver) Acquire(
	ctx context.Context,
) (model.EntityFilterEx, error) {
	return res, nil
}

func (res *resolver) Release(
	ctx context.Context,
) error {
	return nil
}

func (res *resolver) Resolve(
	ctx context.Context,
	rule Rule,
	query []rune,
	weight float64,
	details *Details,
) Hypotheses {
	hs := res.resolve(ctx, rule, query, weight, details)
	// todo: enable sensitivity filter
	//hs2 := hs.filter(details.Sensitivity)

	hs2 := hs
	/*if debug {
		func() {
			docs := res.manager.Documents().Capture()
			defer docs.Release()

			log.Debugf("SEARCH rule=%q, count=%d/%d\n", rule.Name(), len(hs), len(hs2))
			const maxCount = 10
			vs := make(versions, 0, len(hs2))
			items := docs.Items()
			for k, v := range hs2 {
				doc := items[k]
				if doc == nil {
					continue
				}
				vs = append(
					vs,
					&version{
						doc:       doc,
						relevance: v,
						name:      doc.Name,
					},
				)
			}
			sortVersions(vs)
			if len(vs) > maxCount {
				vs = vs[:maxCount]
			}

			list := make(Results, 0, len(vs))
			for _, v := range vs {
				if id, ok := res.docs.Encode(v.doc.Key); ok {
					list = append(
						list,
						&Result{
							Id:        id,
							Relevance: v.relevance,
						},
					)
				}
			}
			result, _ := res.manager.Fetch(ctx, list, model.FetchOptions{Filter: res})
			for _, vv := range result {
				log.Debugf("  SEARCH ITEM, relevance=%g, doc=%s", vv.Relevance, string(vv.Document))
			}
		}()
	}*/

	return hs2
}

func (res *resolver) resolve(
	ctx context.Context,
	rule Rule,
	query []rune,
	weight float64,
	details *Details,
) Hypotheses {
	if details.IsCancel() {
		return nil
	}

	// if qs, ok := res.cache[rule]; ok {
	// 	if hs, ok := qs[string(query)]; ok {
	// 		return hs
	// 	}
	// }

	hs := rule.Search(ctx, res, query, weight, details)
	// if qs, ok := res.cache[rule]; ok {
	// 	qs[string(query)] = hs
	// } else {
	// 	res.cache[rule] = map[string]Hypotheses{string(query): hs}
	// }

	return hs
}

type maxEstimator struct {
}

func (estimator *maxEstimator) Estimate(hs Hypotheses) float64 {
	var relevance float64
	for _, v := range hs {
		if v > relevance {
			relevance = v
		}
	}
	return relevance
}

func NewMaxEstimator() Estimator {
	return new(maxEstimator)
}

type maxMixer struct {
}

func (mixer *maxMixer) Mix(branches []branch) Hypotheses {
	var best branch
	for _, b := range branches {
		if b.relevance > best.relevance {
			best = b
		}
	}

	if best.hypotheses == nil {
		return newHypotheses()
	}

	return best.hypotheses
}

func NewMaxMixer() Mixer {
	return &maxMixer{}
}

type sumMixer struct {
}

func (mixer *sumMixer) Mix(branches []branch) Hypotheses {
	res := newHypotheses()
	for _, b := range branches {
		for k, v := range b.hypotheses {
			if vv, ok := res[k]; ok {
				res[k] = vv + v
			} else {
				res[k] = v
			}
		}
	}

	return res
}

func NewSumMixer() Mixer {
	return &sumMixer{}
}

func newResolver(
	ctx context.Context,
	manager Manager,
	filter EntitiesFilter,
) (Resolver, error) {
	f, err := filter.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("Filter.Acquire: %w", err)
	}

	return &resolver{
		// cache:   make(map[Rule]map[string]Hypotheses, 1024),
		manager:        manager,
		EntityFilterEx: f,
	}, nil
}

func docNameSearchIndexReader(doc *Doc) string {
	return doc.NameSearchIndex
}

func docNameSearchIndexReader2(doc *Doc) string {
	return doc.NameSearchIndex2
}

func docNameSearchIndexReader3(doc *Doc) string {
	return doc.NameSearchIndex3
}

func docNameGroupIndexReader(doc *Doc) string {
	return doc.NameGroupIndex
}

func docInnGroupIndexReader(doc *Doc) string {
	return doc.InnGroupIndex
}

func docBarCodeReader(doc *Doc) string {
	if len(doc.BarCode) == 0 {
		return ""
	}

	barcode := make([]string, len(doc.BarCode))
	copy(barcode, doc.BarCode)
	sort.Strings(barcode)
	return strings.Join(barcode, ";")
}

func docParcelCodeExReader(doc *Doc) string {
	return doc.ParcelCodeEx
}

func docMakerReader(doc *Doc) string {
	return doc.Maker
}

func docInnSearchIndexReader(doc *Doc) string {
	return doc.InnSearchIndex
}

func docInnSearchIndexReader2(doc *Doc) string {
	return doc.InnSearchIndex2
}

func docInnSearchIndexReader3(doc *Doc) string {
	return doc.InnSearchIndex3
}

var debug = true

func DefaultOptions() *Options {
	return &Options{
		Stocks: StockOptions{
			Interval: 60,
			Lang:     Rus,
		},
		Search: *DefaultStrategyOptions(),
	}
}
