package main

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charlievieth/utils/control-completion/cmd/manifest-completion/manifest"
)

const _ = time.ANSIC // WARN: dev only

type NameField struct {
	Name string `json:"name"`
}

type NameFields []NameField

func (n NameFields) Names() []string {
	if len(n) == 0 {
		return nil
	}
	a := make([]string, len(n))
	for i, s := range n {
		a[i] = s.Name
	}
	sort.Strings(a)
	return a
}

type RawManifest struct {
	Containers NameFields `json:"containers"`
	Groups     NameFields `json:"groups"`
}

type AggregatedRawManifest map[string]RawManifest

func (a AggregatedRawManifest) Manifests() []manifest.Manifest {
	all := make([]manifest.Manifest, 0, len(a))
	for name, raw := range a {
		all = append(all, manifest.Manifest{
			Name:       name,
			Containers: raw.Containers.Names(),
			Groups:     raw.Groups.Names(),
		})
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})
	return all
}

// type Manifest struct {
// 	Name       string   `json:"name"`
// 	Containers []string `json:"containers,omitempty"`
// 	Groups     []string `json:"groups,omitempty"`
// }

func LoadAggregatedManifest(name string) ([]manifest.Manifest, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var a AggregatedRawManifest
	if err := json.NewDecoder(f).Decode(&a); err != nil {
		return nil, err
	}
	return a.Manifests(), nil
}

func WriteJSON(filename string, indent bool, v interface{}) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	if indent {
		enc.SetIndent("", "    ")
	}
	return enc.Encode(v)
}

func WriteGob(filename string, v interface{}) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	return enc.Encode(v)
}

var cachedManifest struct {
	manifest.ManifestSlice
	sync.Once
}

func LoadManifestsOnce() ([]manifest.Manifest, error) {
	var err error
	cachedManifest.Do(func() {
		b, e := ioutil.ReadFile("testdata/manifest.proto.raw")
		if e != nil {
			err = e
			return
		}
		err = cachedManifest.Unmarshal(b)
	})
	return cachedManifest.Manifests, err
}

var ErrNoCompletion = errors.New("no completions")

func isLower(c byte) bool { return 'a' <= c && c <= 'z' }
func isDigit(c byte) bool { return '0' <= c && c <= '9' }
func isAlnum(c byte) bool { return isLower(c) || isDigit(c) }

func ValidServicePrefix(s string) bool {
	if len(s) != 0 {
		if !isLower(s[0]) {
			return false
		}
		for i := 0; i < len(s); i++ {
			// if !isAlnum(s[i]) {
			// 	return false
			// }
		}
	}
	return true
}

func AutocompleteService(prefix string) ([]string, error) {
	if strings.Contains(prefix, ".") {
		return nil, errors.New("service name cannot contain '.'")
	}

	ms, err := LoadManifestsOnce()
	if err != nil {
		return nil, err
	}

	// special case
	if prefix == "" {
		a := make([]string, len(ms))
		for i, m := range ms {
			a[i] = m.Name
		}
		return a, nil
	}

	i := sort.Search(len(ms), func(i int) bool {
		return ms[i].Name >= prefix
	})
	if i >= len(ms) || !strings.HasPrefix(ms[i].Name, prefix) {
		return nil, ErrNoCompletion
	}
	a := make([]string, 0, 8)
	for ; i < len(ms); i++ {
		if strings.HasPrefix(ms[i].Name, prefix) {
			a = append(a, ms[i].Name)
		}
	}
	return a, nil
}

// TODO: rename
// "gg.qa" => "gg", "qa"
func SplitNameDot(s string) (name, other string) {
	if n := strings.IndexByte(s, '.'); n != -1 {
		return s[:n], s[n+1:]
	}
	return s, ""
}

func LookupManifest(name string) (*manifest.Manifest, error) {
	ms, err := LoadManifestsOnce()
	if err != nil {
		return nil, err
	}
	i := sort.Search(len(ms), func(i int) bool {
		return ms[i].Name >= name
	})
	if i >= len(ms) || ms[i].Name != name {
		return nil, ErrNoCompletion
	}
	return &ms[i], nil
}

func autocompleteStringSlice(a []string, prefix string) []string {
	if len(a) == 0 {
		return nil
	}
	if prefix == "" {
		return append([]string(nil), a...) // clone
	}
	i := sort.Search(len(a), func(i int) bool {
		return a[i] >= prefix
	})
	if i >= len(a) || !strings.HasPrefix(a[i], prefix) {
		return nil
	}
	var matches []string
	for ; i < len(a); i++ {
		if strings.HasPrefix(a[i], prefix) {
			matches = append(matches, a[i])
		}
	}
	return matches
}

func autocompleteManifest(prefix string, fn func(m *manifest.Manifest) []string) ([]string, error) {
	ndot := strings.Count(prefix, ".") // calm down, it's not DNS
	if ndot == 0 {
		return AutocompleteService(prefix)
	}
	if ndot != 1 {
		return nil, errors.New("multiple dots")
	}

	name, pfx := SplitNameDot(prefix)
	m, err := LookupManifest(name)
	if err != nil {
		return nil, err
	}
	a := autocompleteStringSlice(fn(m), pfx)
	if len(a) == 0 {
		return nil, ErrNoCompletion
	}
	return a, nil
}

func AutocompleteGroup(prefix string) ([]string, error) {
	return autocompleteManifest(prefix, (*manifest.Manifest).GetGroups)
}

func AutocompleteContainer(prefix string) ([]string, error) {
	return autocompleteManifest(prefix, (*manifest.Manifest).GetContainers)
}

// type byPrefix []string
// func (b byPrefix) Len() int {return len(b)}
// func (b byPrefix) Less(i, j int) int {return len(b)}

func main() {
	{
		t := time.Now()
		a, err := AutocompleteGroup("gg.")
		if err != nil {
			Fatal(err)
		}
		d := time.Since(t)
		for _, s := range a {
			fmt.Println(s)
		}
		fmt.Println(d)
		return
	}

	names := []string{
		"chaosdata",
		"chargeback",
		"chat",
		"chromeheadless",
		"ciarchive",
		"ciarchiveagent",
		"ciartifactproxy",
		"cicheckagent",
		"cicostagent",
		"ciexec",
		"ciexecctl",
		"cifleet",
		"cigitagent",
		"ciinfra",
		"cirsyncagent",
		"ciupdateagent",
		"claims",
		"claimsflyte",
	}
	const prefix = "cib"
	n := sort.Search(len(names), func(i int) bool {
		return names[i] >= prefix
	})
	for i, s := range names {
		if i == n {
			fmt.Printf("%d x %s\n", i, s)
		} else {
			fmt.Printf("%d   %s\n", i, s)
		}
	}

	return

}

func EncodeRawManifest(input, output string) error {
	all, err := LoadAggregatedManifest(input)
	if err != nil {
		return err
	}
	x := manifest.ManifestSlice{
		Manifests: make([]manifest.Manifest, len(all)),
	}
	copy(x.Manifests, all)

	b, err := x.Marshal()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(output, b, 0644)
}

/*
	{
		err := EncodeRawManifest(
			"testdata/aggregated_manifest.json",
			"testdata/manifest.proto.raw",
		)
		if err != nil {
			Fatal(err)
		}
		return
	}
*/

/*
	all, err := LoadAggregatedManifest("aggregated_manifest.json")
	if err != nil {
		Fatal(err)
	}

	// fmt.Println("Len:", len(all))
	// fmt.Println("Containers:", len(containers))
	// fmt.Println("Groups:", len(groups))

	// t := time.Now()
	// if err := WriteGob("short.gob", all); err != nil {
	// 	Fatal(err)
	// }
	// fmt.Println(time.Since(t))

	x := manifest.ManifestSlice{
		Manifests: make([]manifest.Manifest, len(all)),
	}
	copy(x.Manifests, all)

	b, err := x.Marshal()
	if err != nil {
		Fatal(err)
	}
	if err := ioutil.WriteFile("testdata/manifest.proto.raw", b, 0644); err != nil {
		Fatal(err)
	}

	{
		t := time.Now()
		var xx manifest.ManifestSlice
		if err := xx.Unmarshal(b); err != nil {
			Fatal(err)
		}
		fmt.Println(time.Since(t))
	}

	// t = time.Now()
	// var x []Manifest
	// if err := gob.NewDecoder(f).Decode(&x); err != nil {
	// 	Fatal(err)
	// }
	// fmt.Println(time.Since(t))

	// f, err := os.Open("aggregated_manifest.json")
	// if err != nil {
	// 	Fatal(err)
	// }
	// defer f.Close()
	// var a AggregatedRawManifest
	// if err := json.NewDecoder(f).Decode(&a); err != nil {
	// 	Fatal(err)
	// }
*/

/*
	containers := make(map[string]int)
	groups := make(map[string]int)
	for _, m := range all {
		for _, s := range m.Containers {
			containers[s]++
		}
		for _, s := range m.Groups {
			groups[s]++
		}
	}
	type NameCount struct {
		Name  string
		Count int
	}
	var cs []NameCount
	for s, n := range containers {
		cs = append(cs, NameCount{s, n})
	}
	sort.Slice(cs, func(i, j int) bool {
		return cs[i].Name < cs[j].Name
	})
	sort.SliceStable(cs, func(i, j int) bool {
		return cs[i].Count < cs[j].Count
	})
	for _, n := range cs {
		fmt.Printf("%d\t%s\n", n.Count, n.Name)
	}
*/

func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	return enc.Encode(v)
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	var format string
	if _, file, line, ok := runtime.Caller(1); ok && file != "" {
		format = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
	} else {
		format = "Error"
	}
	switch err.(type) {
	case error, string:
		fmt.Fprintf(os.Stderr, "%s: %s\n", format, err)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", format, err)
	}
	os.Exit(1)
}
