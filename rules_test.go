package parcels

/*
import (
	"context"
	"fmt"
	"io"
	"sort"
	document2 "spWebFront/FrontKeeper/server/app/domain/service/searcher/document"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchRules(t *testing.T) {
	type Test struct {
		rule  Rule
		cases map[string][]string // Drug name -> list of identifiers of drugs
	}

	var details *Details

	resolver := &resolverMock{}

	docs := map[int64]string{
		1: "Аспирин",
		2: "Анальгин",
		3: "Нокспрей",
		4: "Аскорбинка",
		5: "Аскорбиновая кислота",
	}

	ctx := context.Background()
	ngramRule := NewNgramRule(
		"aaa",
		NewNgramIndex(NgramIndexPositions{}),
		NewNgramParserPrimary(3, NewParserEstimatorPrimary(1)),
	)
	for doc, name := range docs {
		ngramRule.Append(ctx, doc, []rune(name), 1)
	}

	tests := map[string]Test{
		"Ngram": {
			rule: ngramRule,
			cases: map[string][]string{
				"abxyz":  {},
				"Аспир":  {"1"},
				"Аскорб": {"4", "5"},
				"Акорб":  {"4", "5"},
			},
		},
		"Mix": {
			rule: NewMultiRule(
				"mix",
				NewEntries(
					&Entry{Rule: &ruleMock{
						Identifier: Identifier{name: "1"},
						result: map[int64]float32{
							1: 0.1,
							2: 0.2,
							3: 0.3,
						},
					}},
					&Entry{Rule: &ruleMock{
						Identifier: Identifier{name: "2"},
						result: map[int64]float32{
							2: 0.25,
							4: 0.4,
						},
					}},
				),
				NewMaxEstimator(),
				NewSumMixer(),
			),
			cases: map[string][]string{
				"Main": {"2", "4", "3", "1"},
			},
		},
		"Max": {
			rule: NewMaxRule(
				"max",
				NewEntries(
					&Entry{Rule: &ruleMock{
						identifier: identifier{name: "1"},
						result: map[int64]float32{
							1: 0.1,
							2: 0.2,
							3: 0.3,
						},
					}},
					&Entry{
						Rule: &ruleMock{
							identifier: identifier{name: "2"},
							result: map[int64]float32{
								2: 0.25,
								4: 0.4,
							},
						},
					},
				),
			),
			cases: map[string][]string{
				"Main": {"4", "2"},
			},
		},
		"Guard (false)": {
			rule: NewGuardRule(
				"guard",
				nil,
				falsePredicate,
				&ruleMock{
					identifier: identifier{name: "guard"},
					result: map[int64]float32{
						1: 0.1,
						2: 0.2,
						3: 0.3,
					},
				},
			),
			cases: map[string][]string{
				"Primary": {},
			},
		},
		"Guard (true)": {
			rule: NewGuardRule(
				"guard (true)",
				nil,
				truePredicate,
				&ruleMock{
					identifier: identifier{name: "guard (true)"},
					result: map[int64]float32{
						1: 0.1,
						2: 0.2,
						3: 0.3,
					},
				},
			),
			cases: map[string][]string{
				"Primary": {"3", "2", "1"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for query, response := range test.cases {
				t.Run(
					query,
					func(t *testing.T) {
						hs := test.rule.Search(ctx, resolver, []rune(query), details)
						vs := makeDocs(docs, hs)
						sort.Sort(vs)
						actual := make([]string, len(vs))
						for i, v := range vs {
							actual[i] = v.Id
						}
						assert.Equal(t, response, actual)
					},
				)
			}
		})
	}
}

func makeDocs(docs map[int64]string, hs Hypotheses) versions {
	res := make(versions, 0, len(hs))
	for doc, relevance := range hs {
		name := docs[doc]
		res = append(
			res,
			&version{
				Doc: &document2.Doc{
					Id:          fmt.Sprintf("%d", doc),
					SearchIndex: name,
					GroupIndex:  name,
				},
				len:       len(name),
				relevance: relevance,
			},
		)
	}
	return res
}

type resolverMock struct{}

func (r *resolverMock) Resolve(
	ctx context.Context,
	rule Rule,
	query []rune,
	details *Details,
) Hypotheses {
	return rule.Search(ctx, r, query, details)
}

type ruleMock struct {
	Identifier
	result map[int64]float32
}

func (rule *ruleMock) Purge(
	ctx context.Context,
) error {
	return nil
}

func (rule *ruleMock) Search(
	ctx context.Context,
	resolver Resolver,
	query []rune,
	details *Details,
) Hypotheses {
	return rule.result
}

func (rule *ruleMock) Append(
	ctx context.Context,
	id int64,
	name []rune,
) {
}

func (rule *ruleMock) Remove(
	ctx context.Context,
	id int64,
	name []rune,
) {
}

func (rule *ruleMock) Load(r io.Reader) error { return nil }

func (rule *ruleMock) Save(w io.Writer) error { return nil }

func (rule *ruleMock) Log() {

}

func falsePredicate(ctx context.Context, runes []rune) bool {
	return false
}

func truePredicate(ctx context.Context, runes []rune) bool {
	return true
}
*/
