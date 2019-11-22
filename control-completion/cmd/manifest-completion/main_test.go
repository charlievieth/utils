package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/charlievieth/utils/control-completion/cmd/manifest-completion/manifest"
)

func BenchmarkManifestJSON(b *testing.B) {
	data, err := ioutil.ReadFile("testdata/manifest.json")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m []manifest.Manifest
		if err := json.Unmarshal(data, &m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkManifestGob(b *testing.B) {
	data, err := ioutil.ReadFile("testdata/manifest.gob")
	if err != nil {
		b.Fatal(err)
	}
	var buf bytes.Buffer
	buf.Write(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m []manifest.Manifest
		if err := gob.NewDecoder(&buf).Decode(&m); err != nil {
			b.Fatal(err)
		}
		buf.Write(data)
	}
}

func BenchmarkManifestProto(b *testing.B) {
	data, err := ioutil.ReadFile("testdata/manifest.proto.raw")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m manifest.ManifestSlice
		if err := m.Unmarshal(data); err != nil {
			b.Fatal(err)
		}
	}
}
