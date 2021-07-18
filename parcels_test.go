package parcels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"spWebFront/FrontKeeper/infrastructure/bolt"
	"spWebFront/FrontKeeper/infrastructure/core"
	"spWebFront/FrontKeeper/infrastructure/memfile"
	"spWebFront/FrontKeeper/server/app/domain/service/searcher/document"
	"strings"
	"testing"

	"github.com/adverax/echo/generic"

	"github.com/stretchr/testify/require"
)

func getRawFile() ([]map[string]interface{}, error) {
	name := "testdata/drugs.json"
	data, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	var items []map[string]interface{}
	err = core.JsonUnmarshal(data, &items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func makeDrugsFile(items []map[string]interface{}) error {
	name := "testdata/drugs.csv"
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, item := range items {
		val, ok := item["search_index"]
		if !ok {
			continue
		}
		s, ok := core.ConvertToString(val)
		if !ok {
			continue
		}
		_, err := f.WriteString(s + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func readPolling() (map[string][]string, error) {
	dir := "testdata/poll"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	res := make(map[string][]string)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		data, err := ioutil.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		vs := strings.Split(string(data), "\n")
		res[name] = vs
	}

	return res, nil
}

func readDrugs() ([]string, error) {
	name := "testdata/drugs.csv"
	data, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(data), "\n"), nil
}

func TestSearcher(t *testing.T) {
	/*{
		// Convert drugs.json to drugs.csv
		items, err := getRawFile()
		require.NoError(t, err)
		err = makeDrugsFile(items)
		require.NoError(t, err)
	}*/

	// Create database file in memory
	fd, err := memfile.New("drugs", []byte("Hello word"))
	if err != nil {
		log.Fatalf("memfile: %v", err)
	}

	fp := fmt.Sprintf("/proc/self/fd/%d", fd)

	f := os.NewFile(uintptr(fd), fp)
	defer f.Close()

	err = ioutil.WriteFile(fp, []byte{}, 666)
	require.NoError(t, err)

	// Create database, based on virtual file
	db, err := bolt.Open(fp, nil)
	require.NoError(t, err)

	// Create searcher
	documents := document.New()
	searcher, err := newTestSearcher(db, documents)
	require.NoError(t, err)

	// Import raw data
	raw, err := readDrugs()
	require.NoError(t, err)
	var defs []*DocDef
	for i, r := range raw {
		id := fmt.Sprintf("%d", i)
		doc := document.Doc{
			Id:           id,
			Name:         r,
			Drug:         id,
			GroupIndex:   r,
			SearchIndex:  r,
			NameInn:      r,
			InnIndex:     r,
			ParcelCode:   id,
			ParcelCodeEx: id,
			BarCode:      "",
			Weight:       1,
			Analog:       "",
		}
		body, err := json.Marshal(doc)
		require.NoError(t, err)
		defs = append(
			defs,
			&DocDef{
				Head: doc,
				Body: string(body),
			},
		)
	}

	err = searcher.UpdateAll(context.Background(), defs, true)
	require.NoError(t, err)

	// Execute tests
	tests, err := readPolling()
	require.NoError(t, err)

	infinite := float32(len(raw))
	var es []float32
	// ExtendsTuner(true, nil)
	details := new(Details)
	searcher.InitDetails(details)
	details.Tuner = ExtendsTuner(true, nil)
	for query, expected := range tests {
		res, err := searcher.Search(
			context.Background(),
			query,
			"name",
			details,
		)
		require.NoError(t, err)
		actual, err := extractResults(res)
		require.NoError(t, err)
		e := estimateResults(expected, actual, infinite)
		log.Printf("Estimation for %q is %g", query, e)
		es = append(es, e)
	}
	e := estimateEstimations(es)
	t.Error("Average poll distance is ", e)
}

func newTestSearcher(
	db bolt.DB,
	documents document.Manager,
) (Manager, error) {
	options := DefaultStrategyOptions()

	return NewManager(
		db,
		documents,
		DefaultStrategyFactory(
			options,
		),
		ManagerOptions{
			Band: options.Band,
		},
		Rus,
	)
}

func extractResults(res Results) ([]string, error) {
	var rs []string
	for _, r := range res {
		var drug map[string]interface{}
		err := core.JsonUnmarshal(r.Document, &drug)
		if err != nil {
			return nil, err
		}
		val, ok := drug["search_index"]
		if !ok {
			return nil, errors.New("search_index not found")
		}
		name, _ := core.ConvertToString(val)
		rs = append(rs, name)
	}
	return rs, nil
}

func estimateResults(expected, actual []string, infinite float32) float32 {
	var es []float32
	for _, s := range expected {
		e := indexOf(s, actual)
		if e < 0 {
			es = append(es, infinite)
		} else {
			es = append(es, float32(e))
		}
	}
	return estimateEstimations(es)
}

func estimateEstimations(es []float32) float32 {
	return avg(es)

	if len(es) == 0 {
		return 0
	}

	var ee float32
	a := avg(es)
	for _, e := range es {
		x := a - e
		ee += x * x
	}

	return ee / float32(len(es))
}

func indexOf(s string, ss []string) int {
	for i, as := range ss {
		if as == s {
			return i
		}
	}
	return -1
}

func avg(es []float32) float32 {
	if len(es) == 0 {
		return 0
	}

	var sum float32
	for _, e := range es {
		sum += e
	}

	return sum / float32(len(es))
}
