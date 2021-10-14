package dataset

import (
	"errors"
	"strings"
	"testing"
)

func TestReadHazo(t *testing.T) {
	dsEmpty := New("")
	dsOneTaxon := New("")
	dsOneTaxon.CreateTaxon([]int{}, TaxonInit{Id: "t1"})
	dsOneNamedTaxon := New("")
	dsOneNamedTaxon.CreateTaxon([]int{}, TaxonInit{Id: "t1", Name: MultilangText{Scientific: "a"}})
	dsOneNamedCNTaxon := New("")
	dsOneNamedCNTaxon.CreateTaxon([]int{}, TaxonInit{Id: "t1", Name: MultilangText{Scientific: "a", NamesByLangRef: map[string]string{"CN": "米"}}})
	dsTaxonsHierarchy := New("")
	dsTaxonsHierarchy.CreateTaxon([]int{}, TaxonInit{Id: "t1", Name: MultilangText{Scientific: "a"}})
	dsTaxonsHierarchy.CreateTaxon([]int{}, TaxonInit{Id: "t2", Name: MultilangText{Scientific: "b"}})
	dsTaxonsHierarchy.CreateTaxon([]int{1}, TaxonInit{Id: "t3", Name: MultilangText{Scientific: "c"}})
	dsTaxonsHierarchy.CreateTaxon([]int{1}, TaxonInit{Id: "t4", Name: MultilangText{Scientific: "d"}})

	testCases := []struct {
		json string
		ds   *Dataset
		err  error
	}{
		{"", nil, errors.New("")},
		{"{}", dsEmpty, nil},
		{`{ "taxons": [] }`, dsEmpty, nil},
		{`{ "taxons": [ { "id": "t1" } ] }`, dsOneTaxon, nil},
		{`{ "taxons": [ { "id": "t1", "name": "a" } ] }`, dsOneNamedTaxon, nil},
		{`{ "taxons": [ { "id": "t1", "name": "a", "nameCN": "米" } ] }`, dsOneNamedCNTaxon, nil},
		{`{ "taxons": [ { "id": "t1", "name": "a" }, { "id": "t2", "name": "b", "children": ["t3", "t4"] }, { "id": "t3", "name": "c" }, { "id": "t4", "name": "d" } ] }`, dsTaxonsHierarchy, nil},
	}
	for _, testCase := range testCases {
		in := strings.NewReader(testCase.json)
		ds, err := ReadHazo(in)
		if testCase.err == nil && err != nil {
			t.Logf("Unexpected error: %q.", err.Error())
			t.FailNow()
		}
		if testCase.err != nil && err == nil {
			t.Logf("Expected error: %q.", testCase.err.Error())
			t.FailNow()
		}
		if testCase.err != nil {
			continue
		}
		if len(ds.TaxonsById) != len(testCase.ds.TaxonsById) {
			t.Logf("Wrong number of taxons.\nExpected: %d\ngot %d.", len(testCase.ds.TaxonsById), len(ds.TaxonsById))
			t.Fail()
		}
		for id, expectedTaxon := range testCase.ds.TaxonsById {
			taxon, ok := ds.TaxonsById[id]
			if !ok {
				t.Logf("Expected taxon '%s' to exist.", expectedTaxon.Id)
				t.FailNow()
			}
			if expectedTaxon.Id != taxon.Id {
				t.Logf("Wrong taxon id.\nexpected %s\ngot %s", expectedTaxon.Id, taxon.Id)
				t.Fail()
			}
			if !expectedTaxon.Name.Equals(taxon.Name) {
				t.Logf("Wrong taxon name.\nexpected %+v\ngot %+v", expectedTaxon.Name, taxon.Name)
				t.Fail()
			}
			if len(expectedTaxon.Pictures) != len(taxon.Pictures) {
				t.Logf("Wrong number of taxon pictures.\nexpected %d\ngot %d", len(expectedTaxon.Pictures), len(taxon.Pictures))
				t.FailNow()
			}
			for j, expectedPic := range expectedTaxon.Pictures {
				pic := taxon.Pictures[j]
				if expectedPic != pic {
					t.Logf("Wrong picture.\nexpected %+v\ngot %+v", expectedPic, pic)
					t.Fail()
				}
			}
			if len(expectedTaxon.Children) != len(taxon.Children) {
				t.Logf("Wrong number of children for '%s'.\nexpected %d\ngot %d", expectedTaxon.Id, len(expectedTaxon.Children), len(taxon.Children))
				t.FailNow()
			}
			for j, expectedChild := range expectedTaxon.Children {
				child := taxon.Children[j]
				if expectedChild.Id != child.Id {
					t.Logf("Wrong child for '%+s'.\nexpected %+v\ngot %+v", expectedTaxon.Id, expectedChild, child)
					t.Fail()
				}
			}
		}
	}
}
