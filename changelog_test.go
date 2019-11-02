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

package changelog_test

import (
	"bytes"
	"github.com/nekr0z/changelog"
	"os"
	"sort"
	"testing"
	"time"
)

func TestParseMd(t *testing.T) {
	fd, err := os.Open("testdata/ma.md")
	if err != nil {
		t.Fatal(err)
	}
	defer fd.Close()

	cl, err := changelog.ParseMd(fd)
	if err != nil {
		t.Errorf("Parse returned error: %s", err)
	}

	ver := changelog.Version{2, 2, 0, ""}

	if _, ok := cl[ver]; !ok {
		t.Errorf("release 2.2.0 not found")
	}

	d, err := time.Parse("02.01.2006", "21.09.2019")
	if err != nil {
		t.Fatal(err)
	}

	if cl[ver].Date != d {
		t.Errorf("date mismatch: want %v, got %v", d, cl[ver].Date)
	}

	if len(cl[ver].Changes) != 1 {
		t.Fatalf("number of changes mismatch")
	}

	if cl[ver].Changes[0].Type != "Added" || cl[ver].Changes[0].Body != "a way to set custom battery threshold" {
		t.Errorf("got %v", cl[ver].Changes[0])
	}
}

func TestDebian(t *testing.T) {
	var (
		cl changelog.Changelog = map[changelog.Version]changelog.Release{
			changelog.Version{1, 3, 0, ""}: changelog.Release{
				Date: time.Date(2019, 7, 13, 0, 0, 0, 0, time.UTC),
				Changes: []changelog.Change{
					{"Fixed", "some format discrepancies"},
					{"Added", "a useful feature"},
				},
			},
			changelog.Version{1, 3, 1, ""}: changelog.Release{
				Date: time.Date(2019, 7, 18, 0, 0, 0, 0, time.UTC),
				Changes: []changelog.Change{
					{"Fixed", "another bug"},
					{"Fixed", "all the bugs"},
					{"Added", "more features"},
				},
			},
			changelog.Version{1, 3, 1, "rc"}: changelog.Release{
				Date: time.Date(2019, 7, 17, 0, 0, 0, 0, time.UTC),
				Changes: []changelog.Change{
					{"Fixed", "another bug"},
					{"Fixed", "all the bugs"},
					{"Added", "more features"},
				},
			},
		}
		want = []byte(`awesomeapp (1.3.1) stable; urgency=medium

  * Added: more features
  * Fixed: another bug
  * Fixed: all the bugs

 -- John Doe <john@doe.me>  Thu, 18 Jul 2019 00:00:00 +0000

awesomeapp (1.3.1~rc) stable; urgency=medium

  * Added: more features
  * Fixed: another bug
  * Fixed: all the bugs

 -- John Doe <john@doe.me>  Wed, 17 Jul 2019 00:00:00 +0000

awesomeapp (1.3.0) stable; urgency=medium

  * Added: a useful feature
  * Fixed: some format discrepancies

 -- John Doe <john@doe.me>  Sat, 13 Jul 2019 00:00:00 +0000
`)
	)

	for k, rel := range cl {
		rel.Maintainer.Name = "John Doe"
		rel.Maintainer.Email = "john@doe.me"
		cl[k] = rel
	}

	result, err := cl.Debian("awesomeapp")
	if err != nil {
		t.Fatalf("Debian changelog creation failed: %s", err)
	}

	if !bytes.Equal(result, want) {
		t.Errorf("want:\n%s\ngot:\n%s", want, result)
	}

	for k, rel := range cl {
		rel.Urgency = "medium"
		rel.Distribution = "stable"
		cl[k] = rel
	}

	got, err := changelog.ParseDebian(bytes.NewReader(want))
	if err != nil {
		t.Errorf("Error parsing Debian changelog: %s", err)
	}

	if !equal(t, got, cl) {
		t.Errorf(" got: %v\nwant: %v", got, want)
	}
}

func equal(t *testing.T, got, want changelog.Changelog) bool {
	t.Helper()
	if len(got) != len(want) {
		return false
	}
	for v, rel1 := range got {
		rel2, ok := want[v]
		if !ok {
			return false
		}
		if !rel1.Date.Equal(rel2.Date) || rel1.Urgency != rel2.Urgency || rel1.Distribution != rel2.Distribution || rel1.Maintainer.Name != rel2.Maintainer.Name || rel1.Maintainer.Email != rel2.Maintainer.Email {
			return false
		}
		if len(rel1.Changes) != len(rel2.Changes) {
			return false
		}
		sort.SliceStable(rel1.Changes, func(i, j int) bool {
			return rel1.Changes[i].Type < rel1.Changes[j].Type
		})
		sort.SliceStable(rel2.Changes, func(i, j int) bool {
			return rel2.Changes[i].Type < rel2.Changes[j].Type
		})
		for i, ch1 := range rel1.Changes {
			ch2 := rel2.Changes[i]
			if ch1.Type != ch2.Type || ch1.Body != ch2.Body {
				return false
			}
		}
	}
	return true
}

func TestToVersion(t *testing.T) {
	testCases := []struct {
		s string
		v changelog.Version
		e error
	}{
		{"1.1.0", changelog.Version{1, 1, 0, ""}, nil},
		{"51.16.234+14a", changelog.Version{51, 16, 234, ""}, nil},
		{"4.2.15-pre2.11", changelog.Version{4, 2, 15, "pre2.11"}, nil},
		{"1.1.0.2", changelog.Version{0, 0, 0, ""}, changelog.ErrNotSemver},
		{"1.3.-2-15", changelog.Version{0, 0, 0, ""}, changelog.ErrNotSemver},
		{"v3.2.18-rc1+df8891", changelog.Version{0, 0, 0, ""}, changelog.ErrNotSemver},
	}

	for _, testCase := range testCases {
		got, err := changelog.ToVersion(testCase.s)
		if err != testCase.e {
			t.Errorf("want %s, got %s", testCase.e, err)
		} else if got.Major != testCase.v.Major || got.Minor != testCase.v.Minor || got.Patch != testCase.v.Patch || got.Prerelease != testCase.v.Prerelease {
			t.Errorf("want %v, got %v", testCase.v, got)
		}
	}
}
