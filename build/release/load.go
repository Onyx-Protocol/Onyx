package release

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"chain/errors"
)

func load() {
	var f io.ReadCloser
	f, loadErr = os.Open(filepath.Join(os.Getenv(EnvVar), Path))
	if loadErr != nil {
		return
	}
	defer f.Close()
	releases, loadErr = parse(f)
}

var (
	fields  = regexp.MustCompile(`\s+`)
	prodpat = regexp.MustCompile(`^[0-9A-Za-z-]+$`)
)

func parse(r io.Reader) (tab []*Definition, err error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		s := scanner.Text()
		if p := strings.Index(s, "#"); p >= 0 {
			s = s[:p] // strip comments
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		t := fields.Split(s, 5)
		if n := len(t); n != 5 {
			return nil, errors.Wrap(fmt.Errorf("bad record: must have 5 fields (has %d): %s", n, s))
		}

		d := &Definition{
			Product:        t[0],
			Version:        t[1],
			ChainCommit:    t[2],
			ChainprvCommit: t[3],
			Codename:       t[4],
		}

		if !prodpat.MatchString(d.Product) {
			return nil, errors.Wrap(fmt.Errorf("bad product name: %s", d.Product))
		} else if err := CheckVersion(d.Version); err != nil {
			return nil, errors.Wrap(err, d.Product)
		} else if !commitOk(d.ChainCommit) {
			return nil, errors.Wrap(fmt.Errorf("bad commit hash %s", d.ChainCommit))
		} else if !commitOk(d.ChainprvCommit) && d.ChainprvCommit != "n/a" {
			return nil, errors.Wrap(fmt.Errorf("bad commit hash %s", d.ChainprvCommit))
		}
		if d.ChainprvCommit == "n/a" {
			d.ChainprvCommit = ""
		}

		tab = append(tab, d)
	}

	sort.Slice(tab, func(i, j int) bool { return less(tab[i], tab[j]) })

	// Check for duplicates, gaps, point releases
	// across multiple major versions.
	var prev Definition
	for _, d := range tab {
		prevNonRC := prev.Version
		if strings.Contains(prevNonRC, "rc") {
			prevNonRC = Previous(prevNonRC)
		}

		if prev.Product != d.Product {
			prev = *d
			continue
		}
		if prev.Version == d.Version {
			return nil, errors.Wrap(fmt.Errorf("duplicate entry for: %s %s", d.Product, d.Version))
		}
		prev = *d
		if strings.Contains(d.Version, "rc") {
			continue
		}
		if prevNonRC != Previous(d.Version) {
			return nil, errors.Wrap(fmt.Errorf("gap in version sequence before %s %s", d.Product, d.Version))
		}
	}

	return tab, nil
}

func commitOk(s string) bool {
	b, err := hex.DecodeString(s)
	return len(b) == 5 && err == nil
}
