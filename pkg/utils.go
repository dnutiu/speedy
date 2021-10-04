package pkg

import (
	"container/list"
	"fmt"
	"strings"
)

type pair struct {
	key    []string
	isRoot bool
	value  interface{}
}

// FlattenMap flattens a given Map.
func FlattenMap(input map[string]interface{}) *map[string]interface{} {
	var returnValue = make(map[string]interface{})

	var stack = list.New()
	stack.PushBack(pair{key: []string{}, isRoot: true, value: input})

	for {
		element := stack.Back()
		if element == nil {
			break
		}
		stack.Remove(element)

		thePair, ok := element.Value.(pair)
		if !ok {
			panic("can't cast element to pair")
		}

		switch v := thePair.value.(type) {
		// Handle generic maps.
		case map[string]interface{}:
			for key, val := range v {
				if thePair.isRoot {
					stack.PushBack(pair{key: []string{key}, value: val})
				} else {
					newKeys := append(thePair.key, key)
					stack.PushBack(pair{key: newKeys, value: val})
				}
			}
		// Handle arrays.
		case []interface{}:
			lastKey := thePair.key[len(thePair.key)-1]
			for index, val := range v {
				newKeys := make([]string, len(thePair.key))
				itemsCopied := copy(newKeys, thePair.key)
				newKeys[itemsCopied-1] = fmt.Sprintf("%s[%d]", lastKey, index)
				stack.PushBack(pair{key: newKeys, value: val})
			}
		// Handle simple values.
		default:
			returnValue[strings.Join(thePair.key, ".")] = v
		}
	}

	return &returnValue
}
