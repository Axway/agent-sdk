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

	for i, _ := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			tc.init.Update(tc.old, tc.new, tc.idxVal)

			if !reflect.DeepEqual(tc.init, tc.expected) {
				t.Errorf("got: %+v, expected: %+v", tc.init, tc.expected)
			}
		})
	}
}
