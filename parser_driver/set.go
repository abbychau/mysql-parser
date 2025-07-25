// Copyright 2015 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser_driver

import (
	"strconv"
	"strings"

	"github.com/pingcap/errors"
)

var zeroSet = Set{Name: "", Value: 0}

// Set is for MySQL Set type.
type Set struct {
	Name  string
	Value uint64
}

// String implements fmt.Stringer interface.
func (e Set) String() string {
	return e.Name
}

// ToNumber changes Set to float64 for numeric operation.
func (e Set) ToNumber() float64 {
	return float64(e.Value)
}

// Copy deep copy a Set.
func (e Set) Copy() Set {
	return Set{
		Name:  e.Name, // string copy not needed in Go (strings are immutable)
		Value: e.Value,
	}
}

// ParseSet creates a Set with name or value.
func ParseSet(elems []string, name string, collation string) (Set, error) {
	if setName, err := ParseSetName(elems, name, collation); err == nil {
		return setName, nil
	}
	// name doesn't exist, maybe an integer?
	if num, err := strconv.ParseUint(name, 0, 64); err == nil {
		return ParseSetValue(elems, num)
	}

	return Set{}, errors.Errorf("item %s is not in Set %v", name, elems)
}

// ParseSetName creates a Set with name.
func ParseSetName(elems []string, name string, collation string) (Set, error) {
	if len(name) == 0 {
		return zeroSet, nil
	}

	ctor := GetCollator(collation)

	seps := strings.Split(name, ",")
	marked := make(map[string]struct{}, len(seps))
	for _, s := range seps {
		marked[string(ctor.Key(s))] = struct{}{}
	}
	items := make([]string, 0, len(seps))

	value := uint64(0)
	for i, n := range elems {
		key := string(ctor.Key(n))
		if _, ok := marked[key]; ok {
			value |= 1 << uint64(i)
			delete(marked, key)
			items = append(items, n)
		}
	}

	if len(marked) == 0 {
		return Set{Name: strings.Join(items, ","), Value: value}, nil
	}

	return Set{}, errors.Errorf("item %s is not in Set %v", name, elems)
}

var (
	setIndexValue       []uint64
	setIndexInvertValue []uint64
)

func init() {
	setIndexValue = make([]uint64, 64)
	setIndexInvertValue = make([]uint64, 64)

	for i := range 64 {
		setIndexValue[i] = 1 << uint64(i)
		setIndexInvertValue[i] = ^setIndexValue[i]
	}
}

// ParseSetValue creates a Set with special number.
func ParseSetValue(elems []string, number uint64) (Set, error) {
	if number == 0 {
		return zeroSet, nil
	}

	value := number
	var items []string
	for i := range elems {
		if number&setIndexValue[i] > 0 {
			items = append(items, elems[i])
			number &= setIndexInvertValue[i]
		}
	}

	if number != 0 {
		return Set{}, errors.Errorf("invalid number %d for Set %v", number, elems)
	}

	return Set{Name: strings.Join(items, ","), Value: value}, nil
}
