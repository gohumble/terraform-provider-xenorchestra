package internal

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

/*
 * This file contains code that was borrowed from the terraform-provider-aws repo (https://github.com/terraform-providers/terraform-provider-aws). It is provided as is (no modifications) less the code that I did not use.
 *
 * It is covered by the Mozilla Public License Version 2.0.
 *
 * See https://github.com/terraform-providers/terraform-provider-aws/blob/master/LICENSE for copyright and licensing information.
 */

const (
	sentinelIndex = "*"
	sortOrderAsc  = "asc"
	sortOrderDesc = "desc"
)

// TestCheckTypeSetElemNestedAttrs is a resource.TestCheckFunc that accepts a resource
// name, an attribute path, which should use the sentinel value '*' for indexing
// into a TypeSet. The function verifies that an element matches the whole value
// map.
//
// You may check for unset keys, however this will also match keys set to empty
// string. Please provide a map with at least 1 non-empty value.
//
//   map[string]string{
//           "key1": "value",
//       "key2": "",
//   }
//
// Use this function over SDK provided TestCheckFunctions when validating a
// TypeSet where its elements are a nested object with their own attrs/values.
//
// Please note, if the provided value map is not granular enough, there exists
// the possibility you match an element you were not intending to, in the TypeSet.
// Provide a full mapping of attributes to be sure the unique element exists.
func TestCheckTypeSetElemNestedAttrs(name, attr string, values map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		is, err := instanceState(s, name)
		if err != nil {
			return err
		}

		matches := make(map[string]int)
		attrParts := strings.Split(attr, ".")
		if attrParts[len(attrParts)-1] != sentinelIndex {
			return fmt.Errorf("%q does not end with the special value %q", attr, sentinelIndex)
		}
		// account for cases where the user is trying to see if the value is unset/empty
		// there may be ambiguous scenarios where a field was deliberately unset vs set
		// to the empty string, this will match both, which may be a false positive.
		var matchCount int
		for _, v := range values {
			if v != "" {
				matchCount++
			}
		}
		if matchCount == 0 {
			return fmt.Errorf("%#v has no non-empty values", values)
		}
		for stateKey, stateValue := range is.Attributes {
			stateKeyParts := strings.Split(stateKey, ".")
			// a Set/List item with nested attrs would have a flatmap address of
			// at least length 3
			// foo.0.name = "bar"
			if len(stateKeyParts) < 3 {
				continue
			}
			var pathMatch bool
			for i := range attrParts {
				if attrParts[i] != stateKeyParts[i] && attrParts[i] != sentinelIndex {
					break
				}
				if i == len(attrParts)-1 {
					pathMatch = true
				}
			}
			if !pathMatch {
				continue
			}
			id := stateKeyParts[len(attrParts)-1]
			nestedAttr := strings.Join(stateKeyParts[len(attrParts):], ".")
			if v, keyExists := values[nestedAttr]; keyExists && v == stateValue {
				matches[id] = matches[id] + 1
				if matches[id] == matchCount {
					return nil
				}
			}
		}

		return fmt.Errorf("%q no TypeSet element %q, with nested attrs %#v in state: %#v", name, attr, values, is.Attributes)
	}
}

// instanceState returns the primary instance state for the given
// resource name in the root module.
func instanceState(s *terraform.State, name string) (*terraform.InstanceState, error) {
	ms := s.RootModule()
	rs, ok := ms.Resources[name]
	if !ok {
		return nil, fmt.Errorf("Not found: %s in %s", name, ms.Path)
	}

	is := rs.Primary
	if is == nil {
		return nil, fmt.Errorf("No primary instance: %s in %s", name, ms.Path)
	}

	return is, nil
}

func TestCheckTypeSetAttr(nameFirst, keyFirst, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		isFirst, err := instanceState(s, nameFirst)
		if err != nil {
			return err
		}

		return testCheckTypeSetElem(isFirst, keyFirst, value)
	}
}

// TestCheckTypeSetElemAttrPair is a TestCheckFunc that verifies a pair of name/key
// combinations are equal where the first uses the sentinel value to index into a
// TypeSet.
//
// E.g., tfawsresource.TestCheckTypeSetElemAttrPair("aws_autoscaling_group.bar", "availability_zones.*", "data.aws_availability_zones.available", "names.0")
func TestCheckTypeSetElemAttrPair(nameFirst, keyFirst, nameSecond, keySecond string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		isFirst, err := instanceState(s, nameFirst)
		if err != nil {
			return err
		}

		isSecond, err := instanceState(s, nameSecond)
		if err != nil {
			return err
		}

		vSecond, okSecond := isSecond.Attributes[keySecond]
		if !okSecond {
			return fmt.Errorf("%s: Attribute %q not set, cannot be checked against TypeSet", nameSecond, keySecond)
		}

		return testCheckTypeSetElem(isFirst, keyFirst, vSecond)
	}
}

func testCheckTypeSetElem(is *terraform.InstanceState, attr, value string) error {
	attrParts := strings.Split(attr, ".")
	if attrParts[len(attrParts)-1] != sentinelIndex {
		return fmt.Errorf("%q does not end with the special value %q", attr, sentinelIndex)
	}
	for stateKey, stateValue := range is.Attributes {
		if stateValue == value {
			stateKeyParts := strings.Split(stateKey, ".")
			if len(stateKeyParts) == len(attrParts) {
				for i := range attrParts {
					if attrParts[i] != stateKeyParts[i] && attrParts[i] != sentinelIndex {
						break
					}
					if i == len(attrParts)-1 {
						return nil
					}
				}
			}
		}
	}

	return fmt.Errorf("no TypeSet element %q, with value %q in state: %#v", attr, value, is.Attributes)
}

// TestCheckTypeListAttrSorted is a resource.TestCheckFunc that accepts a resource
// name, an attribute path, which should use the sentinel value '*' for indexing
// into a TypeList. The function verifies that the given list is sorted in an ascending
// or descending fashion.
//
// The following invocation would pass with the terraform state given below:
// e.g. internal.TestCheckTypeListAttrSorted("data.xenorchestra_hosts.hosts, "hosts.*.name_label", "asc"),
//
// STATE:
//
// data.xenorchestra_hosts.hosts:
//   ID = 0aea61f4-c9d1-4060-94e8-4eb2024d082c
//   provider = provider.xenorchestra
//   hosts.# = 3
//   hosts.0.name_label = R620-L1
//   hosts.1.name_label = R620-L3
//   hosts.2.name_label = R620-L2
//   sort_by = name_label
//   sort_order = asc
func TestCheckTypeListAttrSorted(name, attr, sortOrder string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		is, err := instanceState(s, name)
		if err != nil {
			return err
		}

		// Over allocate the slice length since
		// matches will be added by index
		matches := make([]string, len(is.Attributes))
		matchCount, sentinelPos := 0, -1
		attrParts := strings.Split(attr, ".")

		for i, attrPart := range attrParts {
			if attrPart == sentinelIndex {
				sentinelPos = i
			}
		}

		if sentinelPos == -1 {
			return fmt.Errorf("%q does not end contain the special value %q", attr, sentinelIndex)
		}

		for stateKey, stateValue := range is.Attributes {
			stateKeyParts := strings.Split(stateKey, ".")
			if len(stateKeyParts) != len(attrParts) {
				continue
			}

			for i := range stateKeyParts {

				if i == sentinelPos {
					continue
				}
				if stateKeyParts[i] != attrParts[i] {
					break
				}

				// Save the match
				if len(attrParts)-1 == i {
					pos, _ := strconv.Atoi(stateKeyParts[sentinelPos])
					matches[pos] = stateValue
					matchCount++
				}

			}
		}

		// Remove the excess uninitilized elements once the relevant
		// attributes are identified due to slice overallocation
		matches = matches[:matchCount-1]
		sorted := sort.SliceIsSorted(matches, func(i, j int) bool {
			switch sortOrder {
			case sortOrderAsc:
				return matches[i] < matches[j]
			case sortOrderDesc:
				return matches[j] < matches[i]
			}
			return false
		})

		if !sorted {
			return errors.New(fmt.Sprintf("expected %v to be sorted %s", matches, sortOrder))
		}

		return nil
	}
}
