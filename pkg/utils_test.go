package pkg

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Test_FlattenMap ensure that FlattenMap works as expected by flattening maps.
func Test_FlattenMap(t *testing.T) {
	var tests = []struct {
		firstMap    map[string]interface{}
		expectedMap map[string]interface{}
	}{
		{
			map[string]interface{}{},
			map[string]interface{}{},
		},
		{
			map[string]interface{}{
				"root": "root_is_always_root",
			},
			map[string]interface{}{
				"root": "root_is_always_root",
			},
		},
		{
			map[string]interface{}{
				"root": map[string]interface{}{
					"testing":     "1",
					"another_key": []int{1, 2, 3},
				},
			},
			map[string]interface{}{
				"root.testing":     "1",
				"root.another_key": []int{1, 2, 3},
			},
		},
		{
			map[string]interface{}{
				"root": map[string]interface{}{
					"list_with_map": []map[string]string{
						{
							"map_in_list": "1",
						},
					},
				},
			},
			map[string]interface{}{
				"root.list_with_map": []map[string]string{
					{
						"map_in_list": "1",
					},
				},
			},
		},
		{
			map[string]interface{}{
				"root": map[string]interface{}{
					"testing": "1",
					"another_key": map[string]interface{}{
						"nested_key":  "nested_value",
						"nested_key2": "nested_value",
						"nested_key3": "nested_value",
					},
				},
			},
			map[string]interface{}{
				"root.testing":                 "1",
				"root.another_key.nested_key":  "nested_value",
				"root.another_key.nested_key2": "nested_value",
				"root.another_key.nested_key3": "nested_value",
			},
		},
		{
			map[string]interface{}{
				"root": map[string]interface{}{
					"testing":     "1",
					"another_key": []interface{}{1, 2, 3},
				},
			},
			map[string]interface{}{
				"root.testing":        "1",
				"root.another_key[0]": 1,
				"root.another_key[1]": 2,
				"root.another_key[2]": 3,
			},
		},
		{
			map[string]interface{}{
				"root": map[string]interface{}{
					"testing":     "1",
					"another_key": []interface{}{[]interface{}{1, 2}, []interface{}{3}},
				},
			},
			map[string]interface{}{
				"root.testing":           "1",
				"root.another_key[0][0]": 1,
				"root.another_key[0][1]": 2,
				"root.another_key[1][0]": 3,
			},
		},
		{
			map[string]interface{}{
				"root": map[string]interface{}{
					"testing": "1",
					"another_key": []interface{}{[]interface{}{1, 2}, []interface{}{map[string]interface{}{
						"inside_list_key":  1,
						"inside_list_key2": []interface{}{"yes", "no"},
					}}},
				},
			},
			map[string]interface{}{
				"root.testing":                               "1",
				"root.another_key[0][0]":                     1,
				"root.another_key[0][1]":                     2,
				"root.another_key[1][0].inside_list_key":     1,
				"root.another_key[1][0].inside_list_key2[0]": "yes",
				"root.another_key[1][0].inside_list_key2[1]": "no",
			},
		},
	}
	for index, tt := range tests {
		t.Run(fmt.Sprintf("test_%d", index), func(t *testing.T) {
			flatMap := FlattenMap(tt.firstMap)
			assert.Equal(t, tt.expectedMap, *flatMap)
		})
	}
}
