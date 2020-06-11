package v1

import (
	"reflect"
	"testing"
)

func TestIndex(t *testing.T) {
	testCases := []struct {
		name     string
		init     index
		old      []string
		new      []string
		idxVal   string
		expected index
	}{
		{
			"add 2 keys",
			index{},
			[]string{},
			[]string{"idx1", "idx2"},
			"val",
			index{
				"idx1": []string{"val"},
				"idx2": []string{"val"},
			},
		},
		{
			"delete 2 keys",
			index{
				"idx1": []string{"val"},
				"idx2": []string{"val"},
			},
			[]string{"idx1", "idx2"},
			[]string{},
			"val",
			index{
				"idx1": []string{},
				"idx2": []string{},
			},
		},
		{
			"delete 1 key, add one key",
			index{
				"idx1": []string{"val"},
				"idx2": []string{"val"},
			},
			[]string{"idx1", "idx2"},
			[]string{"idx2", "idx3"},
			"val",
			index{
				"idx1": []string{},
				"idx2": []string{"val"},
				"idx3": []string{"val"},
			},
		},
		{
			"delete 1 key, add one key, keep ",
			index{
				"idx1": []string{"otherval", "val"},
				"idx2": []string{"val"},
				"idx3": []string{"otherval"},
			},
			[]string{"idx1", "idx2"},
			[]string{"idx2", "idx3"},
			"val",
			index{
				"idx1": []string{"otherval"},
				"idx2": []string{"val"},
				"idx3": []string{"otherval", "val"},
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			tc.init.Update(tc.old, tc.new, tc.idxVal)

			if !reflect.DeepEqual(tc.init, tc.expected) {
				t.Errorf("got: %+v, expected: %+v", tc.init, tc.expected)
			}
		})
	}
}

func TestSets(t *testing.T) {
	opIntersection := func(set1 set, set2 set) set {
		return set1.Intersection(set2)
	}
	opUnion := func(set1 set, set2 set) set {
		return set1.Union(set2)
	}

	testCases := []struct {
		name     string
		set1     set
		set2     set
		op       func(set1 set, set2 set) set
		expected set
	}{{
		"empty intersection with empty",
		set{},
		set{},
		opIntersection,
		set{},
	}, {
		"empty union with empty",
		set{},
		set{},
		opUnion,
		set{},
	}, {
		"empty intersection with one",
		set{},
		newSet("val"),
		opIntersection,
		set{},
	}, {
		"empty union with one",
		set{},
		newSet("val"),
		opUnion,
		newSet("val"),
	}, {
		"identical intersection",
		newSet("val1", "val2"),
		newSet("val1", "val2"),
		opIntersection,
		newSet("val1", "val2"),
	}, {
		"identical union with one",
		newSet("val1", "val2"),
		newSet("val1", "val2"),
		opUnion,
		newSet("val1", "val2"),
	}, {
		"intersection with some common",
		newSet("common1", "common2", "diff_1_1", "diff_1_2"),
		newSet("common1", "common2", "diff_2_1", "diff_2_2"),
		opIntersection,
		newSet("common1", "common2"),
	}, {
		"union with some common",
		newSet("common1", "common2", "diff_1_1", "diff_1_2"),
		newSet("common1", "common2", "diff_2_1", "diff_2_2"),
		opUnion,
		newSet("common1", "common2", "diff_1_1", "diff_1_2", "diff_2_1", "diff_2_2"),
	},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			got := tc.op(tc.set1, tc.set2)

			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("got: %+v, expected: %+v", got, tc.expected)
			}
			got = tc.op(tc.set2, tc.set1)

			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("terms inverted: got: %+v, expected: %+v", got, tc.expected)
			}
		})
	}
}
