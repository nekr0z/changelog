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

// Package changelog provides a way to create, parse and convert changelogs. Currently, only parsing Markdown keep-a-changelog style and Debian changelogs is implemented for input, and only Debian changelog for output.
package changelog

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNotSemver = fmt.Errorf("not semver") // string is not a valid semver
)

// Version is a recognized version (following semver conventions)
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

// ToVersion converts string to Version (if possible)
func ToVersion(s string) (v Version, err error) {
	re := regexp.MustCompile(fmt.Sprintf("^%s$", semver.String()))
	if re.MatchString(s) {
		major, err := strconv.Atoi(semver.ReplaceAllString(s, "${major}"))
		if err == nil {
			minor, err := strconv.Atoi(semver.ReplaceAllString(s, "${minor}"))
			if err == nil {
				patch, err := strconv.Atoi(semver.ReplaceAllString(s, "${patch}"))
				if err == nil {
					pre := semver.ReplaceAllString(s, "${prerelease}")
					v = Version{Major: major, Minor: minor, Patch: patch, Prerelease: pre}
				}
			}
		}
	} else {
		err = ErrNotSemver
	}
	return
}

// Change is one change (usually one line in a changelog)
type Change struct {
	Type string // "Added", "Fixed", etc.
	Body string // the actual description
}

// Release is a single release. A changelog usually comprises of several releases.
type Release struct {
	Date         time.Time
	Changes      []Change
	Urgency      string     // urgency in Debian terms, "medium" is used if none is provided
	Distribution string     // distribution released to (Debian-specific), "stable" is used if none is provided
	Maintainer   Maintainer // package maintainer
}

// Maintainer is the maintainer of the package
type Maintainer struct {
	Name  string
	Email string
}

type Changelog map[Version]Release

var (
	semver = regexp.MustCompile(`(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?`)
	dateRe = regexp.MustCompile(` \d{4}-\d{2}-\d{2}$`)
)

// ParseMd reads a markdown file (keep-a-changelog style format) into Changelog
func ParseMd(r io.Reader) (cl Changelog, err error) {
	cl = make(map[Version]Release)
	scanner := bufio.NewScanner(r)

	var (
		curVer *Version
		curGrp string
	)

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "## "):
			verString := semver.FindString(line)
			var v Version
			v, err = ToVersion(verString)
			if err == nil {
				d, _ := time.Parse(" 2006-01-02", dateRe.FindString(line))
				if _, ok := cl[v]; ok {
					err = fmt.Errorf("multiple releases for %s", verString)
				}
				cl[v] = Release{Date: d}
				curVer = &v
			}
		case strings.HasPrefix(line, "### "):
			curGrp = strings.TrimPrefix(line, "### ")
		case strings.HasPrefix(line, "- ") && curVer != nil:
			rel := cl[*curVer]
			rel.Changes = append(rel.Changes, Change{Type: curGrp, Body: strings.TrimPrefix(line, "- ")})
			cl[*curVer] = rel
		}
	}

	return cl, err
}

// ParseDebian reads a Debian changelog into Changelog
func ParseDebian(r io.Reader) (cl Changelog, err error) {
	cl = make(map[Version]Release)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		verString := semver.FindString(line)
		var v Version
		v, err = ToVersion(verString)
		if err == nil {
			var rel Release
			if _, ok := cl[v]; ok {
				err = fmt.Errorf("multiple releases for %s", verString)
			}
			comps := strings.Split(line, " ")
			for _, comp := range comps {
				switch {
				case strings.HasSuffix(comp, ";"):
					rel.Distribution = strings.TrimSuffix(comp, ";")
				case strings.HasPrefix(comp, "urgency="):
					rel.Urgency = strings.TrimPrefix(comp, "urgency=")
				}
			}
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "  * ") {
					line = strings.TrimPrefix(line, "  * ")
					var chg Change
					s := strings.SplitN(line, ": ", 2)
					if len(s) == 2 {
						chg.Type = s[0]
						chg.Body = s[1]
					} else {
						chg.Body = line
					}
					rel.Changes = append(rel.Changes, chg)
				}
				if strings.HasPrefix(line, " -- ") {
					line = strings.TrimPrefix(line, " -- ")
					s := strings.Split(line, "  ")
					if len(s) != 2 {
						err = fmt.Errorf("can't parse author line for %s", verString)
						break
					}
					var maint Maintainer
					a := strings.Split(s[0], " <")
					if len(a) == 2 {
						maint.Name = a[0]
						maint.Email = strings.TrimSuffix(a[1], ">")
					} else {
						err = fmt.Errorf("error parsing maintainer for %s - no email?", verString)
						maint.Name = s[0]
					}
					rel.Maintainer = maint
					d, er := time.Parse(time.RFC1123Z, s[1])
					if er != nil {
						err = fmt.Errorf("could not parse release date for %s", verString)
						break
					}
					rel.Date = d
					cl[v] = rel
					break
				}
			}
		}
	}
	return
}

// Debian outputs Changelog with debian changelog formatting
func (cl Changelog) Debian(packageName string) (out []byte, err error) {
	type release struct {
		v Version
		d time.Time
	}
	releases := make([]release, 0, len(cl))

	for ver, r := range cl {
		releases = append(releases, release{v: ver, d: r.Date})
	}

	sort.SliceStable(releases, func(i, j int) bool {
		if !releases[i].d.Equal(releases[j].d) {
			return releases[i].d.Before(releases[j].d)
		}
		if releases[i].v.Major != releases[j].v.Major {
			return releases[i].v.Major < releases[j].v.Major
		}
		if releases[i].v.Minor != releases[j].v.Minor {
			return releases[i].v.Minor < releases[j].v.Minor
		}
		if releases[i].v.Patch != releases[j].v.Patch {
			return releases[i].v.Patch < releases[j].v.Patch
		}
		return releases[i].v.Prerelease < releases[j].v.Prerelease
	})

	var s string

	for i := range releases {
		r := releases[len(releases)-i-1]
		rel := cl[r.v]

		if rel.Urgency == "" {
			rel.Urgency = "medium"
		}
		if rel.Distribution == "" {
			rel.Distribution = "stable"
		}

		ver := fmt.Sprintf("%d.%d.%d", r.v.Major, r.v.Minor, r.v.Patch)
		if r.v.Prerelease != "" {
			ver = fmt.Sprintf("%s-%s", ver, r.v.Prerelease)
		}

		s = s + fmt.Sprintf("%s (%s) %s; urgency=%s\n\n", packageName, ver, rel.Distribution, rel.Urgency)

		sort.SliceStable(rel.Changes, func(i, j int) bool {
			return rel.Changes[i].Type < rel.Changes[j].Type
		})

		for _, ch := range rel.Changes {
			s = s + fmt.Sprintf("  * %s: %s\n", ch.Type, ch.Body)
		}

		s = s + fmt.Sprintf("\n -- %s <%s>  %s\n\n", rel.Maintainer.Name, rel.Maintainer.Email, r.d.Format(time.RFC1123Z))
	}

	s = strings.TrimSuffix(s, "\n")
	out = []byte(s)

	return
}
