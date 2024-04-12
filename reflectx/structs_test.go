// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectx

import (
	"reflect"
	"testing"
)

type person struct {
	Name                string `default:"Go Gopher"`
	Age                 int    `default:"35"`
	ProgrammingLanguage string `default:"Go"`
	Pet                 pet
	FavoriteFruit       string `default:"Apple"`
	Data                string `save:"-"`
	OtherPet            *pet
}

type pet struct {
	Name       string
	Type       string `default:"Gopher"`
	Age        int    `default:"7"`
	IsSick     bool
	LikesFoods []string
}

func TestNonDefaultFields(t *testing.T) {
	p := &person{
		Name:                "Go Gopher",
		Age:                 23,
		ProgrammingLanguage: "Go",
		FavoriteFruit:       "Peach",
		Data:                "abcdef",
		Pet: pet{
			Name: "Pet Gopher",
			Type: "Dog",
			Age:  7,
		},
	}
	want := map[string]any{
		"Age":           23,
		"FavoriteFruit": "Peach",
		"Pet": map[string]any{
			"Name": "Pet Gopher",
			"Type": "Dog",
		},
	}
	have := NonDefaultFields(p)
	if !reflect.DeepEqual(have, want) {
		t.Errorf("expected\n%v\n\tbut got\n%v", want, have)
	}
}
