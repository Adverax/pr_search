package parcels

/*
import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"spWebFront/FrontKeeper/infrastructure/core"
	"spWebFront/FrontKeeper/infrastructure/log"
	"time"

	"github.com/adverax/echo/generic"
)

const (
	SearchFormatterEntityId = "search.formatter"
	AnalogFormatterEntityId = "analog.formatter"
	MarginFormatterEntityId = "margin.formatter"
)

type groupItem struct {
	//Id string `json:"id_drug"` // Идентификатор
	//	Parcel             string          `json:"id_parcel`       // Идентификатор партии
	Id                 int64           `json:"-"`
	Name               string          `json:"name_long"` // Полное название (вместе с производителем)
	Group              string          `json:"group"`
	NameSimple         string          `json:"name_simple"`    // Простое название
	NameGroupIndex     string          `json:"group_index"`    // Строка для группировки товара (название без производителя)
	NameSortIndex      string          `json:"sort_index"`     // Строка для сортировки по имени
	InnSearchIndex     string          `json:"name_inn"`       // Полное название действующего вещества
	InnGroupIndex      string          `json:"inn_index"`      // Строка для группировки товара по inn (название без производителя)
	InnSortIndex       string          `json:"inn_sort_index"` // Строка для сортировки по ИНН
	Margin             float32         `json:"margin"`         // Маржа
	ProvisorBonus      float32         `json:"provisor_bonus"` // Бонус провизора
	DateExpire         int64           `json:"date_expire"`    // Дата истечения
	BrandFormName      string          `json:"brand_form_name"`
	BrandCompAmountSum float32         `json:"brand_comp_amount_sum"`
	BrandPackVolume    float32         `json:"brand_pack_volume"`
	UnitPackName       string          `json:"unit_pack_name`
	Number             int64           `json:"number"`
	Document           json.RawMessage `json:"-"` // Сам документ
	Relevance          float64         `json:"-"` // Релевантность элемента
}

type groupItems []*groupItem

type groupItemsNameSorter struct {
	items groupItems
}

func (sorter *groupItemsNameSorter) Len() int {
	return len(sorter.items)
}

func (sorter *groupItemsNameSorter) Swap(i, j int) {
	sorter.items[i], sorter.items[j] = sorter.items[j], sorter.items[i]
}

func (sorter *groupItemsNameSorter) Less(i, j int) bool {
	a := sorter.items[i]
	b := sorter.items[j]
	if a.Margin > b.Margin {
		return true
	}
	if a.Margin < b.Margin {
		return false
	}
	if a.ProvisorBonus > b.ProvisorBonus {
		return true
	}
	if a.ProvisorBonus < b.ProvisorBonus {
		return false
	}
	return a.DateExpire > b.DateExpire
}

type groupItemsInnSorter struct {
	items groupItems
}

func (sorter *groupItemsInnSorter) Len() int {
	return len(sorter.items)
}

func (sorter *groupItemsInnSorter) Swap(i, j int) {
	sorter.items[i], sorter.items[j] = sorter.items[j], sorter.items[i]
}

func (sorter *groupItemsInnSorter) Less(i, j int) bool {
	a := sorter.items[i]
	b := sorter.items[j]
	if a.Margin > b.Margin {
		return true
	}
	if a.Margin < b.Margin {
		return false
	}
	if a.ProvisorBonus > b.ProvisorBonus {
		return true
	}
	if a.ProvisorBonus < b.ProvisorBonus {
		return false
	}
	return a.DateExpire > b.DateExpire
}

type groupItemsMainSorter struct {
	items groupItems
}

func (sorter *groupItemsMainSorter) Len() int {
	return len(sorter.items)
}

func (sorter *groupItemsMainSorter) Swap(i, j int) {
	sorter.items[i], sorter.items[j] = sorter.items[j], sorter.items[i]
}

func (sorter *groupItemsMainSorter) Less(i, j int) bool {
	a := sorter.items[i]
	b := sorter.items[j]
	if a.Relevance > b.Relevance {
		return true
	}
	if a.Relevance < b.Relevance {
		return false
	}
	c := core.Compare(a.NameSimple, b.NameSimple)
	if c < 0 {
		return true
	}
	if c > 0 {
		return false
	}
	c = core.Compare(a.BrandFormName, b.BrandFormName)
	if c < 0 {
		return true
	}
	if c > 0 {
		return false
	}
	if a.BrandCompAmountSum > b.BrandCompAmountSum {
		return true
	}
	if a.BrandCompAmountSum < b.BrandCompAmountSum {
		return false
	}
	if a.BrandPackVolume > b.BrandPackVolume {
		return true
	}
	if a.BrandPackVolume < b.BrandPackVolume {
		return false
	}
	c = core.Compare(a.UnitPackName, b.UnitPackName)
	if c < 0 {
		return true
	}
	if c > 0 {
		return false
	}
	return a.Number > b.Number
}

type groupItemsMainInnSorter struct {
	items groupItems
}

func (sorter *groupItemsMainInnSorter) Len() int {
	return len(sorter.items)
}

func (sorter *groupItemsMainInnSorter) Swap(i, j int) {
	sorter.items[i], sorter.items[j] = sorter.items[j], sorter.items[i]
}

func (sorter *groupItemsMainInnSorter) Less(i, j int) bool {
	a := sorter.items[i]
	b := sorter.items[j]
	c := core.Compare(a.BrandFormName, b.BrandFormName)
	if c < 0 {
		return true
	}
	if c > 0 {
		return false
	}
	if a.BrandCompAmountSum > b.BrandCompAmountSum {
		return true
	}
	if a.BrandCompAmountSum < b.BrandCompAmountSum {
		return false
	}
	if a.BrandPackVolume > b.BrandPackVolume {
		return true
	}
	if a.BrandPackVolume < b.BrandPackVolume {
		return false
	}
	c = core.Compare(a.UnitPackName, b.UnitPackName)
	if c < 0 {
		return true
	}
	if c > 0 {
		return false
	}
	return a.Number > b.Number
}

type simpleFloatRow struct {
	Id        int64
	Name      string
	Relevance float64
	Document  json.RawMessage
}

type simpleFloatSorter struct {
	items []*simpleFloatRow
}

func (sorter *simpleFloatSorter) Len() int {
	return len(sorter.items)
}

func (sorter *simpleFloatSorter) Swap(i, j int) {
	sorter.items[i], sorter.items[j] = sorter.items[j], sorter.items[i]
}

func (sorter *simpleFloatSorter) Less(i, j int) bool {
	a := sorter.items[i]
	b := sorter.items[j]
	if a.Relevance < b.Relevance {
		return false
	}
	if a.Relevance > b.Relevance {
		return true
	}
	return core.Compare(a.Name, b.Name) < 0
}

type simpleIntRow struct {
	Id        string
	Name      string
	Relevance int64
	Document  json.RawMessage
}

type simpleIntSorter struct {
	items []*simpleIntRow
}

func (sorter *simpleIntSorter) Len() int {
	return len(sorter.items)
}

func (sorter *simpleIntSorter) Swap(i, j int) {
	sorter.items[i], sorter.items[j] = sorter.items[j], sorter.items[i]
}

func (sorter *simpleIntSorter) Less(i, j int) bool {
	a := sorter.items[i]
	b := sorter.items[j]
	if a.Relevance < b.Relevance {
		return false
	}
	if a.Relevance > b.Relevance {
		return true
	}
	return core.Compare(a.Name, b.Name) < 0
}

type group struct {
	Name               string     `json:"name"`
	NameSimple         string     `json:"name_simple"`
	BrandFormName      string     `json:"brand_form_name"`
	BrandCompAmountSum float32    `json:"brand_comp_amount"`
	BrandPackVolume    float32    `json:"brand_pack_volume"`
	UnitPackName       string     `json:"unit_pack_name`
	Sort               string     `json:"inn_sort_index"`
	Relevance          float64    `json:"relevance"`
	Items              groupItems `json:"items"`
}

type groups []*group

type groupsNameSorter struct {
	items groups
}

func (sorter *groupsNameSorter) Len() int {
	return len(sorter.items)
}

func (sorter *groupsNameSorter) Swap(i, j int) {
	sorter.items[i], sorter.items[j] = sorter.items[j], sorter.items[i]
}

func (sorter *groupsNameSorter) Less(i, j int) bool {
	a := sorter.items[i]
	b := sorter.items[j]
	if a.Relevance > b.Relevance {
		return true
	}
	if a.Relevance < b.Relevance {
		return false
	}
	return core.Compare(a.Name, b.Name) < 0
}

type groupsNameSorter2 struct {
	items groups
}

func (sorter *groupsNameSorter2) Len() int {
	return len(sorter.items)
}

func (sorter *groupsNameSorter2) Swap(i, j int) {
	sorter.items[i], sorter.items[j] = sorter.items[j], sorter.items[i]
}

func (sorter *groupsNameSorter2) Less(i, j int) bool {
	a := sorter.items[i]
	b := sorter.items[j]
	return core.Compare(a.Name, b.Name) < 0
}

type groupsMainSorter struct {
	items groups
}

func (sorter *groupsMainSorter) Len() int {
	return len(sorter.items)
}

func (sorter *groupsMainSorter) Swap(i, j int) {
	sorter.items[i], sorter.items[j] = sorter.items[j], sorter.items[i]
}

func (sorter *groupsMainSorter) Less(i, j int) bool {
	a := sorter.items[i]
	b := sorter.items[j]
	if a.Relevance > b.Relevance {
		return true
	}
	if a.Relevance < b.Relevance {
		return false
	}
	c := core.Compare(a.NameSimple, b.NameSimple)
	if c < 0 {
		return true
	}
	if c > 0 {
		return false
	}
	c = core.Compare(a.BrandFormName, b.BrandFormName)
	if c < 0 {
		return true
	}
	if c > 0 {
		return false
	}
	if a.BrandCompAmountSum > b.BrandCompAmountSum {
		return true
	}
	if a.BrandCompAmountSum < b.BrandCompAmountSum {
		return false
	}
	if a.BrandPackVolume > b.BrandPackVolume {
		return true
	}
	if a.BrandPackVolume < b.BrandPackVolume {
		return false
	}
	c = core.Compare(a.UnitPackName, b.UnitPackName)
	if c < 0 {
		return true
	}
	return false
}

type groupsInnSorter struct {
	items groups
}

func (sorter *groupsInnSorter) Len() int {
	return len(sorter.items)
}

func (sorter *groupsInnSorter) Swap(i, j int) {
	sorter.items[i], sorter.items[j] = sorter.items[j], sorter.items[i]
}

func (sorter *groupsInnSorter) Less(i, j int) bool {
	a := sorter.items[i]
	b := sorter.items[j]
	return core.Compare(a.Sort, b.Sort) < 0
}

// Formatter is abstract formatter Results
type Formatter interface {
	Format(ctx context.Context, docs Results) (Results, error)
}

type formatterUnsorted struct {
}

func (formatter *formatterUnsorted) Format(
	ctx context.Context,
	items Results,
) (Results, error) {
	res := make(Results, 0, len(items))
	for _, item := range items {
		if item.Document == nil {
			continue
		}

		drug := make(map[string]interface{})
		err := core.JsonUnmarshal(item.Document, &drug)
		if err != nil {
			continue
		}

		if value, ok := drug["group_index"]; ok {
			if val, ok := core.ConvertToString(value); ok {
				val += fmt.Sprintf("; %d", item.Id)
				drug["group_index"] = val
			}
		}

		data, err := json.Marshal(drug)
		if err != nil {
			continue
		}

		res = append(
			res,
			&Result{
				Id:        item.Id,
				Relevance: item.Relevance,
				Document:  data,
			},
		)
	}

	return res, nil
}

// Форматирование по топу маржи.
type formatterMargin struct {
}

func (formatter *formatterMargin) Format(
	ctx context.Context,
	items Results,
) (Results, error) {
	list := make([]*simpleFloatRow, 0, len(items))
	for _, item := range items {
		var row struct {
			Name   string  `json:"name_long"`
			Margin float32 `json:"margin"`
		}
		err := core.JsonUnmarshal(item.Document, &row)
		if err != nil {
			return nil, fmt.Errorf("Unmarshal: %w", err)
		}
		list = append(
			list,
			&simpleFloatRow{
				Id:        item.Id,
				Name:      row.Name,
				Relevance: float64(row.Margin),
				Document:  item.Document,
			},
		)
	}

	sort.Sort(&simpleFloatSorter{list})

	res := make(Results, len(list))
	for i, row := range list {
		res[i] = &Result{
			Id:        row.Id,
			Relevance: row.Relevance,
			Document:  row.Document,
		}
	}

	return res, nil
}

// Форматирование по названию, производителю, сроку годности.
type formatterMakerExpire struct {
}

func (formatter *formatterMakerExpire) Format(
	ctx context.Context,
	items Results,
) (Results, error) {
	list := make([]*simpleFloatRow, 0, len(items))
	for _, item := range items {
		var row struct {
			Name   string `json:"name_simple"`
			Maker  string `json:"maker"`
			Expire int64  `json:"parcel_date_expire,omitempty"`
		}
		err := core.JsonUnmarshal(item.Document, &row)
		if err != nil {
			//log.Println("RAW", string(item.Document))
			return nil, fmt.Errorf("Unmarshal: %w", err)
		}

		var expire string
		if row.Expire > 0 {
			expire = time.Unix(int64(row.Expire), 0).String()
		}

		list = append(list, &simpleFloatRow{
			Id:        item.Id,
			Name:      row.Name + " / " + row.Maker + " / " + expire,
			Relevance: float64(item.Relevance),
			Document:  item.Document,
		})
	}

	sort.Sort(&simpleFloatSorter{list})

	res := make(Results, len(list))
	for i, row := range list {
		res[i] = &Result{
			Id:        row.Id,
			Relevance: row.Relevance,
			Document:  row.Document,
		}
	}

	return res, nil
}

// Группирует по полю group_index.
// Карждую группу сортирует по полю margin в порядке убывания.
// Все группы сортирует в порядке убывания их релевантности.
type formatterWithGrouping struct {
}

func (formatter *formatterWithGrouping) Format(
	ctx context.Context,
	items Results,
) (
	Results,
	error,
) {
	value := ctx.Value("innMode")
	innMode, _ := value.(bool)

	gs := make(map[string]*group, 256)
	for _, item := range items {
		row := new(groupItem)
		err := core.JsonUnmarshal(item.Document, &row)
		if err != nil {
			log.Printf("Document %s has errors %s\n", item.Id, err.Error())
			continue
		}
		row.Relevance = item.Relevance
		row.Document = item.Document
		if innMode {
			row.Group = row.InnGroupIndex
		} else {
			row.Group = row.NameGroupIndex
		}
		relevance := item.Relevance
		if g, ok := gs[row.Group]; ok {
			if g.Relevance < item.Relevance {
				g.Relevance = item.Relevance
			}
			g.Items = append(g.Items, row)
		} else {
			g := &group{
				Name:      row.Group,
				Sort:      row.InnSortIndex,
				Relevance: relevance,
				Items:     groupItems{row},
			}
			// if innMode {
			// 	g.Name = row.NameSortIndex
			// } else {
			// 	g.Name = row.Group
			// }
			gs[g.Name] = g
		}
	}

	// Группировка
	groups := make(groups, 0, len(gs))
	for _, g := range gs {
		row := g.Items[0]

		if innMode {
			sort.Sort(&groupItemsMainInnSorter{items: g.Items})
			g.NameSimple = ""
		} else {
			sort.Sort(&groupItemsMainSorter{items: g.Items})
			g.NameSimple = row.NameSimple
		}

		g.BrandFormName = row.BrandFormName
		g.BrandCompAmountSum = row.BrandCompAmountSum
		g.BrandPackVolume = row.BrandPackVolume
		g.UnitPackName = row.UnitPackName
		g.Relevance = row.Relevance

		groups = append(groups, g)
	}

	sort.Sort(&groupsMainSorter{items: groups})

	// Слияние отсортированных групп
	res := make(Results, 0, 256)
	for _, group := range groups {
		// log.Println("GROUP ------------ ", group.NameSimple, ": ", group.Relevance)
		for _, item := range group.Items {
			//			log.Println("ITEM", item.NameSimple, item.BrandFormName, item.BrandCompAmountSum, item.BrandPackVolume, item.UnitPackName, item.Number)
			res = append(
				res,
				&Result{
					Id:        item.Id,
					Relevance: item.Relevance,
					Document:  item.Document,
				},
			)
		}
	}

	return res, nil
}

type formatterWithoutGrouping struct {
}

func (formatter *formatterWithoutGrouping) Format(
	ctx context.Context,
	items Results,
) (
	Results,
	error,
) {
	list := make(groupItems, 0, len(items))
	for _, d := range items {
		doc := new(groupItem)
		err := core.JsonUnmarshal(d.Document, doc)
		if err != nil {
			// todo: log error
			continue
		}
		doc.Document = d.Document
		doc.Relevance = d.Relevance
		list = append(list, doc)
	}
	sort.Sort(&groupItemsNameSorter{items: list})
	result := make(Results, len(list))
	for i, d := range list {
		result[i] = &Result{
			Id:        d.Id,
			Relevance: d.Relevance,
			Document:  d.Document,
		}
	}
	return result, nil
}

type capacityFormatter struct {
	capacity int
}

func (formatter *capacityFormatter) Format(
	ctx context.Context,
	items Results,
) (Results, error) {
	if len(items) < formatter.capacity {
		return items, nil
	}

	return items[:formatter.capacity], nil
}

// NewGroupFormatter is constructor for create instance of formatter
func NewGroupFormatter() Formatter {
	return &formatterWithGrouping{}
}

// NewWithoutGroupFormatter is constructor for create instace of formatter
func NewWithoutGroupFormatter() Formatter {
	return &formatterWithoutGrouping{}
}

// NewUnsortedFormatter is constructor for create formatter without formatting.
func NewUnsortedFormatter() Formatter {
	return &formatterUnsorted{}
}

func NewMarginFormatter() Formatter {
	return new(formatterMargin)
}

func NewMakerFormatter() Formatter {
	return new(formatterMakerExpire)
}

func NewCapacityFormatter(
	capacity int,
) Formatter {
	return &capacityFormatter{
		capacity: capacity,
	}
}
*/
