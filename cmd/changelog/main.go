// Copyright (C) 2021 Evgeny Kuznetsov (evgeny@kuznetsov.md)
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
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// The changelog binary generates a Debian changelog from a Markdown keep-a-changelog-style one
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nekr0z/changelog"
)

const usageText = `changelog is a tool for converting keep-a-changelog-style changelog to Debian changelog.
Usage:
	changelog [flags] <filename>
Flags:
`

var (
	name  = flag.String("n", "Maintainer", "Maintainer name")
	email = flag.String("e", "maintainer@example.com", "Maintainer email")
	pack  = flag.String("p", "package", "package name")
)

func main() {
	flag.Parse()
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usageText)
		flag.PrintDefaults()
	}

	fn := flag.Arg(0)
	if fn == "" {
		flag.Usage()
		return
	}

	f, err := os.Open(fn)
	if err != nil {
		fatalf("could not open file: %v", err)
	}

	defer f.Close()

	cl, err := changelog.ParseMd(f)
	if err != nil {
		fatalf("could not parse changelog: %v", err)
	}

	for v, rel := range cl {
		rel.Maintainer = changelog.Maintainer{Name: *name, Email: *email}
		cl[v] = rel
	}

	b, err := cl.Debian(*pack)
	if err != nil {
		fatalf("could not convert changelog to Debian format: %v", err)
	}

	fout, err := os.Create("debian.changelog")
	if err != nil {
		fatalf("could not create debian.changelog: %v", err)
	}
	defer fout.Close()
	_, err = fout.Write(b)
	if err != nil {
		fatalf("could not write Debian changelog: %v", err)
	}
	err = fout.Sync()
	if err != nil {
		fatalf("error: %v", err)
	}
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
