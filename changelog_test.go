// Copyright (C) 2019 Evgeny Kuznetsov (evgeny@kuznetsov.md)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along tihe this program. If not, see <https://www.gnu.org/licenses/>.

package changelog

import (
	"testing"
	"time"
)

func TestSortReleases(t *testing.T) {
	test := []string{"5.2.14", "5.2.14-rc1", "6.1.12", "5.2.15", "5.3.0-rc", "5.2.14-rc2", "5.3.0"}
	want := []string{"5.2.14-rc1", "5.2.14-rc2", "5.2.14", "5.2.15", "5.3.0-rc", "5.3.0", "6.1.12"}

	date := time.Now()

	var releases []release

	for _, ver := range test {
		v, err := ToVersion(ver)
		if err != nil {
			t.Fatal(err)
		}
		rel := release{v, date}
		releases = append(releases, rel)
	}

	sortReleases(releases)

	var got []string

	for _, rel := range releases {
		got = append(got, rel.v.String())
	}

	if len(want) != len(got) {
		t.Fatalf("want %s\ngot %s", want, got)
	}

	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("want %s\ngot %s", want, got)
		}
	}
}
