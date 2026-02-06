// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"reflect"
	"testing"
)

func TestStructToEnvVar(t *testing.T) {
	inProg := false
	type Place struct {
		StreetName string `env:"street_name,omitempty"`
		Zipcode    string `env:"zip"`
	}
	type Owner struct {
		FirstName   string `env:"firstName"`
		AlsoIgnored string
		Address     Place    `                  nestedEnvPrefix:"addr_"`
		Age         uint     `env:"age"`
		Phones      []string `env:"phone"`
		InProgress  *bool    `env:"in_progress"`
	}
	type FavoriteFood struct {
		Type string `env:"type"`
		Qty  uint   `env:"qty"`
	}
	type Animal struct {
		Name               string `env:"name"`
		NbrLegs            int    `env:"nbr_legs"`
		IgnoreField        string
		Owner              Owner          `               nestedEnvPrefix:"OWNER_"`
		FavoriteFoods      []FavoriteFood `nestedEnvPrefix:"fav_food"`
		OtherFavoriteFoods []FavoriteFood `nestedEnvPrefix:"fav_food"`
	}
	type TestCase struct {
		desc    string
		srcData Animal
		want    []string
	}
	testCases := []TestCase{
		{
			"merge non empty EnvVar on Pod",
			Animal{
				Name:        "john_Doe",
				NbrLegs:     20,
				IgnoreField: "no tag",
				Owner: Owner{
					FirstName:  "enrico",
					Age:        39,
					Phones:     []string{"001", "002"},
					InProgress: &inProg,
				},
			},
			[]string{
				"PGBACKREST_name=john_Doe",
				"PGBACKREST_nbr_legs=20",
				"PGBACKREST_OWNER_firstName=enrico",
				"PGBACKREST_OWNER_age=39",
				"PGBACKREST_OWNER_phone1=001",
				"PGBACKREST_OWNER_phone2=002",
				"PGBACKREST_OWNER_in_progress=n",
			},
		},
		{
			"merge non empty EnvVar on Pod",
			Animal{
				Name:        "john_Doe",
				NbrLegs:     20,
				IgnoreField: "no tag",
				Owner: Owner{
					FirstName: "enrico",
					Age:       39,
					Phones:    []string{"001"},
				},
			},
			[]string{
				"PGBACKREST_name=john_Doe",
				"PGBACKREST_nbr_legs=20",
				"PGBACKREST_OWNER_firstName=enrico",
				"PGBACKREST_OWNER_age=39",
				"PGBACKREST_OWNER_phone1=001",
			},
		},
		{
			"merge non empty EnvVar on Pod",
			Animal{
				Name:        "john_Doe",
				NbrLegs:     20,
				IgnoreField: "no tag",
				Owner: Owner{
					FirstName: "enrico",
					Age:       39,
					Phones:    []string{"001"},
				},
				FavoriteFoods: []FavoriteFood{
					{
						Type: "raw",
						Qty:  2,
					},
					{
						Type: "hard",
						Qty:  3,
					},
				},
				OtherFavoriteFoods: []FavoriteFood{
					{
						Type: "barf",
						Qty:  12,
					},
				},
			},
			[]string{
				"PGBACKREST_name=john_Doe",
				"PGBACKREST_nbr_legs=20",
				"PGBACKREST_OWNER_firstName=enrico",
				"PGBACKREST_OWNER_age=39",
				"PGBACKREST_OWNER_phone1=001",
				"PGBACKREST_fav_food1type=raw",
				"PGBACKREST_fav_food1qty=2",
				"PGBACKREST_fav_food2type=hard",
				"PGBACKREST_fav_food2qty=3",
				"PGBACKREST_fav_food3type=barf",
				"PGBACKREST_fav_food3qty=12",
			},
		},
	}

	for _, tc := range testCases {
		f := func(t *testing.T) {
			r, _ := StructToEnvVars(tc.srcData, "PGBACKREST_")
			if !reflect.DeepEqual(r, tc.want) {
				t.Errorf("Expected %v, but got %v", tc.want, r)
			}
		}
		t.Run(tc.desc, f)
	}
}
