package parcels

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"spWebFront/FrontKeeper/infrastructure/core"
	"spWebFront/FrontKeeper/server/app/domain/model"
	"spWebFront/FrontKeeper/server/app/domain/repository"
)


type DocManagerEx interface {
	DocManager
	ForEach(
		ctx context.Context,
		action func(ctx context.Context, doc *Doc) error,
	) error
}

type mapDocManager struct {
	items   map[int64]*Doc
	parcels repository.ParcelRepository
}

func (docs *mapDocManager) Purge(
	ctx context.Context,
) error {
	docs.items = make(map[int64]*Doc, 1024)
	return nil
}

func (docs *mapDocManager) Remove(
	ctx context.Context,
	id int64,
) error {
	delete(docs.items, id)
	return nil
}

func (docs *mapDocManager) Append(
	ctx context.Context,
	doc *Doc,
) error {
	docs.items[doc.Id] = doc
	return nil
}

func (docs *mapDocManager) Versions(
	ctx context.Context,
	hs Hypotheses,
	reader Reader,
) (versions, error) {
	//threshold := 1 - tolerance
	i := 0
	vs := make(versions, len(hs))
	for k, v := range hs {
		doc := docs.items[k]
		// if v < threshold {
		// 	continue
		// }
		group := reader(doc)
		vs[i] = &version{
			doc:       doc,
			relevance: v,
			len:       len(group),
			group:     group,
			name:      doc.NameLong, // todo: lang???
		}
		i++
	}
	if i < len(hs) {
		vs = vs[:i]
	}
	sortVersions(vs)
	return vs, nil
}

func (docs *mapDocManager) Resolve(
	ctx context.Context,
	hs Hypotheses,
	details *Details,
	reader Reader,
) (res model.Parcels, err error) {
	filter, err := details.Filter.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("Filter.Acquire: %w", err)
	}
	defer filter.Release(ctx)

	vs, err := docs.Versions(ctx, hs, reader)
	if err != nil {
		return nil, fmt.Errorf("Versions: %w", err)
	}
	res = make(model.Parcels, 0, len(vs))
	band := &details.Band
	var start float64
	var prev float64

	for _, v := range vs {
		doc, err := docs.parcels.Find(ctx, v.doc.Id)
		if err != nil {
			return nil, fmt.Errorf("Find: %w", err)
		}
		if doc == nil {
			continue
		}

		document, err := docs.filter(
			ctx,
			doc.Document,
			filter,
			v.doc.Id,
			v.relevance,
		)
		if err != nil {
			// todo: log error
			continue
		}
		if document == "" {
			continue
		}

		ln := len(res)
		if ln >= band.Capacity {
			break
		}

		if v.relevance < band.Threshold {
			break
		}

		prev = v.relevance
		if ln == 0 {
			start = v.relevance
		} else {
			ratioRel := v.relevance / prev
			ratioAbs := v.relevance / start
			if ratioRel < band.Diff.Rel || ratioAbs < band.Diff.Abs {
				break
			}
		}

		parcel := new(model.Parcel)
		err = core.JsonUnmarshal([]byte(document), &parcel)
		if err != nil {
			return nil, fmt.Errorf("Unmarshal: %w", err)
		}
		parcel.Document = document
		parcel.Relevance = v.relevance
		res = append(res, parcel)
	}

	return res, nil
}

func (docs *mapDocManager) filter(
	ctx context.Context,
	document string,
	filter model.EntityFilter,
	id int64,
	relevance float64,
) (string, error) {
	modify := debug && id != 0

	if filter == nil && !modify {
		return document, nil
	}

	var attrs map[string]interface{}
	err := core.JsonUnmarshal([]byte(document), &attrs)
	if err != nil {
		return "", fmt.Errorf("Unmarshal: %w", err)
	}

	if filter != nil {
		err := filter.Filter(ctx, id, attrs)
		if err != nil {
			if core.Cause(err) == core.ErrAbort {
				return "", nil
			}
			return "", fmt.Errorf("Filter: %w", err)
		}
	}

	if modify {
		attrs[".id"] = id
		attrs[".relevance"] = relevance
	}

	data, err := json.Marshal(attrs)
	if err != nil {
		return "", fmt.Errorf("Marshal: %w", err)
	}
	return string(data), nil
}

func (docs *mapDocManager) ForEach(
	ctx context.Context,
	action func(ctx context.Context, doc *Doc) error,
) error {
	for _, doc := range docs.items {
		err := action(ctx, doc)
		if err != nil {
			if err == core.ErrBreak {
				break
			}
			return fmt.Errorf("action: %w", err)
		}
	}
	return nil
}

func NewMapDocManager(
	parcels repository.ParcelRepository,
) DocManagerEx {
	return &mapDocManager{
		items:   make(map[int64]*Doc, 1024),
		parcels: parcels,
	}
}

type repositoryDocManager struct {
	parcels repository.ParcelRepository
}

func (docs *repositoryDocManager) Purge(
	ctx context.Context,
) error {
	return nil
}

func (docs *repositoryDocManager) Remove(
	ctx context.Context,
	id int64,
) error {
	return nil
}

func (docs *repositoryDocManager) Append(
	ctx context.Context,
	doc *Doc,
) error {
	return nil
}

func (docs *repositoryDocManager) Resolve(
	ctx context.Context,
	hs Hypotheses,
	details *Details,
	reader Reader,
) (res model.Parcels, err error) {
	ps, err := docs.parcels.FindByHypotheses(
		ctx,
		hs,
		model.ParcelBandOptions{
			Filter: model.ParcelBandFilter{
				Corp:  true,
				Store: true,
			},
			Sort:  details.Sort,
			Capacity: details.Band.Capacity,
			Lang:  details.Lang,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("FindByHypotheses: %w", err)
	}
	return sortParcels(ps)
}

func NewRepositoryDocManager(
	parcels repository.ParcelRepository,
) DocManager {
	return &repositoryDocManager{
		parcels: parcels,
	}
}

func sortParcels(
	parcels model.Parcels,
) (res model.Parcels, err error) {
	us := make(map[string]*union, 1024)
	unions := make([]*union, 0, 1024)

	for _, p := range parcels {
		name := strings.ToLower(p.NameGroupIndex)
		if u, ok := us[name]; ok {
			u.items = append(u.items, p)
			if u.relevance < p.Relevance {
				u.relevance = p.Relevance
			}
		} else {
			u := &union{
				name:      name,
				items:     model.Parcels{p},
				relevance: p.Relevance,
			}
			us[name] = u
			unions = append(unions, u)
		}
	}

	sorter := &unionSorter{
		items:   unions,
		epsilon: 0.01 * float64(minUnionRelevance(unions)) / float64(len(unions)),
	}

	sort.Sort(sorter)

	res = make(model.Parcels, 0, len(parcels))
	for _, u := range unions {
		res = append(res, u.items...)
	}

	return res, nil
}

type union struct {
	name      string
	relevance float64
	items     []*model.Parcel
}

type unionSorter struct {
	items   []*union
	epsilon float64
}

func (us *unionSorter) Len() int {
	return len(us.items)
}

func (us *unionSorter) Swap(i, j int) {
	us.items[i], us.items[j] = us.items[j], us.items[i]
}

func (us *unionSorter) Less(i, j int) bool {
	a := us.items[i]
	b := us.items[j]

	if us.isEqual(a.relevance, b.relevance) {
		return core.Compare(a.name, b.name) < 0
	}

	return a.relevance > b.relevance
}

func (us *unionSorter) isEqual(a, b float64) bool {
	delta := math.Abs(a - b)
	return delta < us.epsilon
}

func minUnionRelevance(us []*union) float64 {
	var min float64 = 1000000
	for _, u := range us {
		if u.relevance < min {
			min = u.relevance
		}
	}
	return min
}
