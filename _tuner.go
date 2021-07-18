package parcels
/*
import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"hash"
	"math"
	"sort"
	"spWebFront/FrontKeeper/infrastructure/core"
	"spWebFront/FrontKeeper/infrastructure/log"
	"spWebFront/FrontKeeper/server/app/domain/model"
)

type Parcel = model.Parcel
/*struct {
	model.Parcel
	Relevance          float64 `json:"-"`                     // Релевантность
	Group              string  `json:"group"`                 // Группа
	NameSimple         string  `json:"name_simple"`           // Простое название
	NameSearchIndex    string  `json:"search_index"`          // Строка для поиска товара (язык 1)
	NameSearchIndex2   string  `json:"search_index_lang2"`    // Строка для поиска товара (язык 2)
	NameSearchIndex3   string  `json:"search_index_lang3"`    // Строка для поиска товара (язык 3)
	NameGroupIndex     string  `json:"group_index"`           // Строка для группировки товара (название без производителя)
	InnGroupIndex      string  `json:"inn_index"`             // Строка для группировки товара по inn (название без производителя)
	InnSearchIndex     string  `json:"name_inn"`              // Полное название действующего вещества (язык 1)
	InnSearchIndex2    string  `json:"name_inn_lang2"`        // Полное название действующего вещества (язык 2)
	InnSearchIndex3    string  `json:"name_inn_lang3"`        // Полное название действующего вещества (язык 3)
	InnSortIndex       string  `json:"inn_sort_index"`        // Строка для сортировки по ИНН
	ProvisorBonus      float64 `json:"provisor_bonus"`        // Бонус провизора
	DateExpire         int64   `json:"date_expire"`           // Срок годности медикамента
	BrandFormName      string  `json:"brand_form_name"`       // Название бренда
	BrandCompAmountSum float64 `json:"brand_comp_amount_sum"` //
	BrandPackVolume    float64 `json:"brand_pack_volume"`     // Объем фасовки
	UnitPackName       string  `json:"unit_pack_name`         // Единицы измерения фасовки
	Number             int64   `json:"number"`                //
	// NameSortIndex      string  `json:"sort_index"`         // Строка для сортировки по имени
}* /

type Parcels []*Parcel

func (ps Parcels) Relevance() (relevance float64) {
	for i, p := range ps {
		if i == 0 || p.Relevance > relevance {
			relevance = p.Relevance
		}
	}
	return
}

func (ps Parcels) ToDocuments() []json.RawMessage {
	res := make([]json.RawMessage, 0, len(ps))
	for _, p := range ps {
		if p == nil {
			continue
		}
		res = append(res, core.CloneBytes([]byte(p.Document)))
	}
	return res
}

type TunerOptions struct {
	Sort     []*model.Sort `json:"sort"`
	Group    []string      `json:"group"`
	Distinct []string      `json:"distinct"`
	Lang     int           `json:"lang"`
}

type Tuner interface {
	Execute(parcels Parcels, options TunerOptions) Parcels
}

type ParcelSorter interface {
	Compile(fields []*model.Sort, lang int) ParcelHandler
	Execute(parcels Parcels, comparer ParcelComparer) Parcels
}

type ParcelDistincter interface {
	Compile(fields []string, lang int) ParcelHandler
	Execute(parcels Parcels, hasher ParcelHasher) Parcels
}

type ParcelGrouper interface {
	Compile(fields []string, lang int) ParcelHandler
	Execute(parcels Parcels, hasher ParcelHasher) map[string]Parcels
}

type mainTuner struct {
	grouper    ParcelGrouper
	sorter     ParcelSorter
	distincter ParcelDistincter
}

func (tuner *mainTuner) Execute(
	parcels Parcels,
	options TunerOptions,
) Parcels {
	if len(parcels) <= 1 {
		return parcels
	}

	// if len(sort) == 0 {
	// 	sort = []*model.Sort{
	// 		{
	// 			Field: "margin",
	// 			Desc:  true,
	// 		},
	// 	}
	// }

	// if len(distincts) == 0 {
	// 	distincts = []string{"id_drug"}
	// }

	groups := tuner.grouper.Compile(options.Group, options.Lang)
	distincts := tuner.distincter.Compile(options.Distinct, options.Lang)
	sorts := tuner.sorter.Compile(options.Sort, options.Lang)
	gs := tuner.grouper.Execute(parcels, groups)
	for i, parcels := range gs {
		parcels = tuner.sorter.Execute(parcels, sorts)
		parcels = tuner.distincter.Execute(parcels, distincts)
		gs[i] = parcels
	}

	gs2 := sortGroups(gs, compileNameReader(options.Lang))

	result := make(Parcels, 0, len(parcels))
	for _, g := range gs2 {
		for _, p := range g.parcels {
			result = append(result, p)
		}
	}

	return result
}

func NewTuner(
	grouper ParcelGrouper,
	sorter ParcelSorter,
	distincter ParcelDistincter,
) Tuner {
	return &mainTuner{
		grouper:    grouper,
		sorter:     sorter,
		distincter: distincter,
	}
}

func sortGroups(
	groups map[string]Parcels,
	reader ParcelStringReader,
) parcelGroups {
	gs := make(parcelGroups, 0, len(groups))
	for _, g := range groups {
		if len(g) == 0 {
			continue
		}
		gs = append(
			gs,
			&parcelGroup{
				relevance: g.Relevance(),
				parcels:   g,
				name:      reader(g[0]),
			},
		)
	}
	sort.Sort(gs)

	// for _, g := range gs {
	// 	log.Println("GROUP", g.name, g.relevance)
	// 	for _, p := range g.parcels {
	// 		log.Println("PARCEL", p.Name)
	// 	}
	// }

	return gs
}

func compileNameReader(
	lang int,
) ParcelStringReader {
	switch lang {
	case 2:
		return parcelName2Reader
	case 3:
		return parcelName3Reader
	default:
		return parcelNameReader
	}
}

func compileRowHandler(
	field string,
	handler ParcelHandler,
	lang int,
) ParcelHandler {
	switch field {
	case "relevance":
		return &floatHandler{
			reader:        parcelRelevanceReader,
			ParcelHandler: handler,
		}

	case "margin":
		return &floatHandler{
			reader:        parcelMarginReader,
			ParcelHandler: handler,
		}

	case "price", "price_sell_sum":
		return &floatHandler{
			reader:        parcelPriceReader,
			ParcelHandler: handler,
		}

	case "quantity":
		return &floatHandler{
			reader:        parcelQuantityReader,
			ParcelHandler: handler,
		}

	case "date_start", "date":
		return &intHandler{
			reader:        parcelDateStartReader,
			ParcelHandler: handler,
		}

	case "id_drug":
		return &stringHandler{
			reader:        parcelDrugReader,
			ParcelHandler: handler,
		}

	case "group":
		return &stringHandler{
			reader:        parcelGroupReader,
			ParcelHandler: handler,
		}

	case "name_simple":
		return &stringHandler{
			reader:        parcelNameSimpleReader,
			ParcelHandler: handler,
		}

	case "name_group_index":
		return &stringHandler{
			reader:        parcelNameGroupIndexReader,
			ParcelHandler: handler,
		}

	case "inn_group_index":
		return &stringHandler{
			reader:        parcelInnGroupIndexReader,
			ParcelHandler: handler,
		}

	case "inn_sort_index":
		return &stringHandler{
			reader:        parcelInnSortIndexReader,
			ParcelHandler: handler,
		}

	case "provisor_bonus":
		return &floatHandler{
			reader:        parcelProvisorBonusReader,
			ParcelHandler: handler,
		}

	case "date_expire":
		return &intHandler{
			reader:        parcelDateExpireReader,
			ParcelHandler: handler,
		}

	case "brand_forrm_name":
		return &stringHandler{
			reader:        parcelBrandFormNameReader,
			ParcelHandler: handler,
		}

	case "brand_comp_amount_sum":
		return &floatHandler{
			reader:        parcelBrandCompAmountSumReader,
			ParcelHandler: handler,
		}

	case "brand_pack_volume":
		return &floatHandler{
			reader:        parcelBrandPackVolumeReader,
			ParcelHandler: handler,
		}

	case "unit_pack_name":
		return &stringHandler{
			reader:        parcelUnitPackNameReader,
			ParcelHandler: handler,
		}

	case "number":
		return &intHandler{
			reader:        parcelNumberReader,
			ParcelHandler: handler,
		}

	case "search_index_lang1":
		return &stringHandler{
			reader:        parcelNameSearchIndexReader,
			ParcelHandler: handler,
		}

	case "search_index_lang2":
		return &stringHandler{
			reader:        parcelNameSearchIndex2Reader,
			ParcelHandler: handler,
		}

	case "search_index_lang3":
		return &stringHandler{
			reader:        parcelNameSearchIndex2Reader,
			ParcelHandler: handler,
		}

	case "name", "name_long_lang1":
		return &stringHandler{
			reader:        parcelNameReader,
			ParcelHandler: handler,
		}

	case "name2", "name_long_lang2":
		return &stringHandler{
			reader:        parcelName2Reader,
			ParcelHandler: handler,
		}

	case "name3", "name_long_lang3":
		return &stringHandler{
			reader:        parcelName3Reader,
			ParcelHandler: handler,
		}

	case "inn_search_index", "name_inn_lang1":
		return &stringHandler{
			reader:        parcelInnSearchIndexReader,
			ParcelHandler: handler,
		}

	case "name_inn_lang2":
		return &stringHandler{
			reader:        parcelInnSearchIndex2Reader,
			ParcelHandler: handler,
		}

	case "name_inn_lang3":
		return &stringHandler{
			reader:        parcelInnSearchIndex3Reader,
			ParcelHandler: handler,
		}

	case "search_index":
		switch lang {
		case 2:
			return compileRowHandler("search_index_lang2", handler, lang)
		case 3:
			return compileRowHandler("search_index_lang3", handler, lang)
		default:
			return compileRowHandler("search_index_lang1", handler, lang)
		}

	case "name_long":
		switch lang {
		case 2:
			return compileRowHandler("name_long_lang2", handler, lang)
		case 3:
			return compileRowHandler("name_long_lang3", handler, lang)
		default:
			return compileRowHandler("name_long_lang1", handler, lang)
		}

	case "name_inn_lang":
		switch lang {
		case 2:
			return compileRowHandler("name_inn_lang2", handler, lang)
		case 3:
			return compileRowHandler("name_inn_lang3", handler, lang)
		default:
			return compileRowHandler("name_inn_lang1", handler, lang)
		}

	default:
		log.Println("Unknown parcel's field", field)
		return handler
	}
}

type ParcelIntReader func(parcel *Parcel) int64

type ParcelFloatReader func(parcel *Parcel) float64

type ParcelStringReader func(parcel *Parcel) string

var (
	parcelEmptyReader = func(parcel *Parcel) string {
		return ""
	}

	parcelDrugReader = func(parcel *Parcel) string {
		return parcel.Drug
	}

	parcelRelevanceReader = func(parcel *Parcel) float64 {
		return parcel.Relevance
	}

	parcelMarginReader = func(parcel *Parcel) float64 {
		return parcel.Margin
	}

	parcelPriceReader = func(parcel *Parcel) float64 {
		return parcel.Price
	}

	parcelQuantityReader = func(parcel *Parcel) float64 {
		if parcel.QuantDiv == 0 {
			return float64(parcel.QuantNum)
		}
		return float64(parcel.QuantNum) / float64(parcel.QuantDiv)
	}

	parcelDateStartReader = func(parcel *Parcel) int64 {
		return parcel.DateStart
	}

	parcelNameReader = func(parcel *Parcel) string {
		return parcel.Name
	}

	parcelName2Reader = func(parcel *Parcel) string {
		return parcel.Name2
	}

	parcelName3Reader = func(parcel *Parcel) string {
		return parcel.Name3
	}

	parcelGroupReader = func(parcel *Parcel) string {
		return parcel.Group
	}

	parcelNameSimpleReader = func(parcel *Parcel) string {
		return parcel.NameSimple
	}

	parcelNameGroupIndexReader = func(parcel *Parcel) string {
		return parcel.NameGroupIndex
	}

	parcelInnGroupIndexReader = func(parcel *Parcel) string {
		return parcel.InnGroupIndex
	}

	parcelInnSearchIndexReader = func(parcel *Parcel) string {
		return parcel.InnSearchIndex
	}

	parcelInnSearchIndex2Reader = func(parcel *Parcel) string {
		return parcel.InnSearchIndex2
	}

	parcelInnSearchIndex3Reader = func(parcel *Parcel) string {
		return parcel.InnSearchIndex3
	}

	parcelInnSortIndexReader = func(parcel *Parcel) string {
		return parcel.InnSortIndex
	}

	parcelProvisorBonusReader = func(parcel *Parcel) float64 {
		return parcel.ProvisorBonus
	}

	parcelDateExpireReader = func(parcel *Parcel) int64 {
		return parcel.DateExpire
	}

	parcelBrandFormNameReader = func(parcel *Parcel) string {
		return parcel.BrandFormName
	}

	parcelBrandCompAmountSumReader = func(parcel *Parcel) float64 {
		return parcel.BrandCompAmountSum
	}

	parcelBrandPackVolumeReader = func(parcel *Parcel) float64 {
		return parcel.BrandPackVolume
	}

	parcelUnitPackNameReader = func(parcel *Parcel) string {
		return parcel.UnitPackName
	}

	parcelNumberReader = func(parcel *Parcel) int64 {
		return parcel.Number
	}

	parcelNameSearchIndexReader = func(parcel *Parcel) string {
		return parcel.NameSearchIndex
	}

	parcelNameSearchIndex2Reader = func(parcel *Parcel) string {
		return parcel.NameSearchIndex2
	}

	parcelNameSearchIndex3Reader = func(parcel *Parcel) string {
		return parcel.NameSearchIndex3
	}
)

type ParcelHasher interface {
	Hash(parcel *Parcel, hash hash.Hash)
}

type ParcelComparer interface {
	Compare(a, b *Parcel) int
}

type ParcelHandler interface {
	ParcelComparer
	ParcelHasher
}

type descHandler struct {
	ParcelHandler
}

func (h *descHandler) Compare(a, b *Parcel) int {
	val := h.ParcelHandler.Compare(a, b)
	return -val
}

type intHandler struct {
	ParcelHandler
	reader ParcelIntReader
}

func (h *intHandler) Compare(a, b *Parcel) int {
	va := h.reader(a)
	vb := h.reader(b)
	if va < vb {
		return -1
	}
	if va > vb {
		return 1
	}
	return h.ParcelHandler.Compare(a, b)
}

func (h *intHandler) Hash(parcel *Parcel, hash hash.Hash) {
	value := h.reader(parcel)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, uint64(value))
	hash.Write(bytes)
	h.ParcelHandler.Hash(parcel, hash)
}

type floatHandler struct {
	ParcelHandler
	reader ParcelFloatReader
}

func (h *floatHandler) Compare(a, b *Parcel) int {
	va := h.reader(a)
	vb := h.reader(b)
	if va < vb {
		return -1
	}
	if va > vb {
		return 1
	}
	return h.ParcelHandler.Compare(a, b)
}

func (h *floatHandler) Hash(parcel *Parcel, hash hash.Hash) {
	value := h.reader(parcel)
	bits := math.Float64bits(value)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bits)
	hash.Write(bytes)
	h.ParcelHandler.Hash(parcel, hash)
}

type stringHandler struct {
	ParcelHandler
	reader ParcelStringReader
}

func (h *stringHandler) Compare(a, b *Parcel) int {
	va := h.reader(a)
	vb := h.reader(b)
	cmp := core.Compare(vb, va)
	if cmp != 0 {
		return cmp
	}
	return h.ParcelHandler.Compare(a, b)
}

func (h *stringHandler) Hash(parcel *Parcel, hash hash.Hash) {
	value := h.reader(parcel)
	bytes := []byte(value)
	hash.Write(bytes)
	h.ParcelHandler.Hash(parcel, hash)
}

type terminalHandler struct {
	reader ParcelStringReader
}

func (h *terminalHandler) Compare(a, b *Parcel) int {
	va := h.reader(a)
	vb := h.reader(b)
	return core.Compare(vb, va)
}

func (h *terminalHandler) Hash(parcel *Parcel, hash hash.Hash) {
}

func NewTerminal() ParcelHandler {
	return &terminalHandler{
		reader: parcelEmptyReader,
	}
}

type parcelGroup struct {
	relevance float64
	parcels   Parcels
	name      string
}

type parcelGroups []*parcelGroup

func (groups parcelGroups) Len() int {
	return len(groups)
}

func (groups parcelGroups) Swap(i, j int) {
	groups[i], groups[j] = groups[j], groups[i]
}

func (groups parcelGroups) Less(i, j int) bool {
	a := groups[i]
	b := groups[j]
	if a.relevance < b.relevance {
		return false
	}
	if a.relevance > b.relevance {
		return true
	}
	return core.Compare(a.name, b.name) < 0
}

type parcelSorterHelper struct {
	comparer ParcelComparer
	parcels  Parcels
}

func (helper *parcelSorterHelper) Len() int {
	return len(helper.parcels)
}

func (helper *parcelSorterHelper) Swap(i, j int) {
	helper.parcels[i], helper.parcels[j] = helper.parcels[j], helper.parcels[i]
}

func (helper *parcelSorterHelper) Less(i, j int) bool {
	return helper.comparer.Compare(helper.parcels[i], helper.parcels[j]) < 0
}

type parcelSorter struct {
	terminal ParcelHandler
	disabled bool
}

func (sorter *parcelSorter) Compile(
	fields []*model.Sort,
	lang int,
) ParcelHandler {
	if len(fields) == 0 {
		return sorter.terminal
	}

	handler := sorter.Compile(fields[1:], lang)

	s := fields[0]
	h := compileRowHandler(s.Field, handler, lang)
	if s.Desc {
		return &descHandler{ParcelHandler: h}
	}

	return h
}

func (sorter *parcelSorter) Execute(
	parcels Parcels,
	comparer ParcelComparer,
) Parcels {
	if sorter.disabled || len(parcels) <= 1 {
		return parcels
	}

	helper := &parcelSorterHelper{
		parcels:  parcels,
		comparer: comparer,
	}
	sort.Sort(helper)
	return helper.parcels
}

func NewParcelSorter(
	terminal ParcelHandler,
	enabled bool,
) ParcelSorter {
	return &parcelSorter{
		terminal: terminal,
		disabled: !enabled,
	}
}

type parcelDistincter struct {
	terminal ParcelHandler
	disabled bool
}

func (dist *parcelDistincter) Compile(
	fields []string,
	lang int,
) ParcelHandler {
	if len(fields) == 0 {
		return dist.terminal
	}

	handler := dist.Compile(fields[1:], lang)
	return compileRowHandler(fields[0], handler, lang)
}

func (dist *parcelDistincter) Execute(
	parcels Parcels,
	hasher ParcelHasher,
) Parcels {
	if dist.disabled || len(parcels) <= 1 {
		return parcels
	}

	var res Parcels
	items := make(map[string]bool, len(parcels))
	for _, parcel := range parcels {
		hash := md5.New()
		hasher.Hash(parcel, hash)
		key := hex.EncodeToString(hash.Sum(nil))

		if _, exists := items[key]; exists {
			continue
		}

		items[key] = true
		res = append(res, parcel)
	}
	return res
}

func NewParcelDistinctrer(
	terminal ParcelHandler,
	enabled bool,
) ParcelDistincter {
	return &parcelDistincter{
		terminal: terminal,
		disabled: !enabled,
	}
}

type parcelGrouper struct {
	terminal ParcelHandler
	disabled bool
}

func (grouper *parcelGrouper) Compile(
	fields []string,
	lang int,
) ParcelHandler {
	if len(fields) == 0 {
		return grouper.terminal
	}

	handler := grouper.Compile(fields[1:], lang)
	return compileRowHandler(fields[0], handler, lang)
}

func (grouper *parcelGrouper) Execute(
	parcels Parcels,
	hasher ParcelHasher,
) map[string]Parcels {
	if grouper.disabled || len(parcels) <= 1 {
		return map[string]Parcels{"": parcels}
	}

	gs := make(map[string]Parcels, 128)
	for _, parcel := range parcels {
		hash := md5.New()
		hasher.Hash(parcel, hash)
		key := hex.EncodeToString(hash.Sum(nil))
		if g, ok := gs[key]; ok {
			gs[key] = append(g, parcel)
		} else {
			gs[key] = Parcels{parcel}
		}
	}

	return gs
}

func NewParcelGrouper(
	terminal ParcelHandler,
	enabled bool,
) ParcelGrouper {
	return &parcelGrouper{
		terminal: terminal,
		disabled: !enabled,
	}
}
*/