// Package jsonc benchmarks for parse, serialize, format, and path operations
package jsonc

import (
	"testing"
)

// ---- Benchmark helpers ----

var (
	benchSmallConfig = `{"xkeen":{"speed_balancer":{"enabled":true,"interval":10}}}`
	benchSmallDoc    = func() *Node {
		d, _ := Parse(benchSmallConfig)
		return d
	}()

	benchLargeConfig = func() string {
		b := `{`
		for i := 0; i < 100; i++ {
			if i > 0 {
				b += ","
			}
			b += `"key_` + intStr(i) + `":` + `{"value":` + intStr(i) + `}`
		}
		b += `}`
		return b
	}()

	benchLargeDoc = func() *Node {
		d, _ := Parse(benchLargeConfig)
		return d
	}()

	benchJSONCConfig = `{
  // XKeen global config
  "xkeen": {
    "speed_balancer": {
      // Enable speed balancing across multiple outbounds
      "enabled": false,
      "interval": 10,
      "log": true
    },
    "routing": {
      "mode": "smart",
      "dns": "8.8.8.8"
    }
  }
}`

	benchJSONCDoc = func() *Node {
		d, _ := Parse(benchJSONCConfig)
		return d
	}()

	benchDeepPath = "xkeen.routing"
	benchSetPath  = "xkeen.routing.mode"
	benchDeepKey  = "xkeen"
)

// ---- Parse benchmarks ----

func BenchmarkParse_Small(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Parse(benchSmallConfig)
	}
}

func BenchmarkParse_Large(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Parse(benchLargeConfig)
	}
}

func BenchmarkParse_JSONC(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Parse(benchJSONCConfig)
	}
}

// ---- Serialize benchmarks ----

func BenchmarkSerialize_Small(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Serialize(benchSmallDoc)
	}
}

func BenchmarkSerialize_Large(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Serialize(benchLargeDoc)
	}
}

func BenchmarkSerialize_JSONC(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Serialize(benchJSONCDoc)
	}
}

// ---- Format benchmarks ----

func BenchmarkFormat_Small(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Format(benchSmallDoc, nil)
	}
}

func BenchmarkFormat_Large(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Format(benchLargeDoc, nil)
	}
}

func BenchmarkFormat_JSONC(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Format(benchJSONCDoc, nil)
	}
}

// ---- Parse+Format round-trip benchmark ----

func BenchmarkParseFormat_JSONC(b *testing.B) {
	for i := 0; i < b.N; i++ {
		doc, err := Parse(benchJSONCConfig)
		if err != nil {
			b.Fatal(err)
		}
		_ = Format(doc, nil)
	}
}

// ---- GetPath benchmark ----

func BenchmarkGetPath_Deep(b *testing.B) {
	obj := benchJSONCDoc.Root()
	for i := 0; i < b.N; i++ {
		_ = obj.GetPath(benchDeepPath)
	}
}

func BenchmarkGetPath_Shallow(b *testing.B) {
	obj := benchJSONCDoc.Root()
	for i := 0; i < b.N; i++ {
		_ = obj.GetPath(benchDeepKey)
	}
}

// ---- SetPath benchmark ----

func BenchmarkSetPath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		obj := benchJSONCDoc.Root()
		obj.SetPath(benchSetPath, "new_value")
	}
}

// ---- Get benchmark (direct) ----

func BenchmarkGet_Direct(b *testing.B) {
	obj := benchJSONCDoc.Root()
	for i := 0; i < b.N; i++ {
		_ = obj.Get(benchDeepKey)
	}
}

// ---- Builder benchmark ----

func BenchmarkBuilder_Object(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Object(
			"name", "test",
			"value", 42,
			"enabled", true,
		)
	}
}

func BenchmarkBuilder_Array(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Array(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	}
}

// ---- Object/Array constructor benchmarks ----

func BenchmarkNewObject_10Fields(b *testing.B) {
	for i := 0; i < b.N; i++ {
		o := NewObject()
		for j := 0; j < 10; j++ {
			o.AppendChild(NewMember(intStr(j), NewNull()))
		}
		_ = o
	}
}

// ---- UnmarshalPath benchmark ----

func BenchmarkUnmarshalPath(b *testing.B) {
	obj := benchJSONCDoc.Root()
	var dst testUnmarshalTarget
	for i := 0; i < b.N; i++ {
		_ = obj.UnmarshalPath(benchDeepPath, &dst)
	}
}

// ---- MarshalPath benchmark ----

func BenchmarkMarshalPath(b *testing.B) {
	obj := benchJSONCDoc.Root()
	src := testUnmarshalTarget{
		Enabled:  true,
		Name:     "test",
		Interval: 42,
	}
	for i := 0; i < b.N; i++ {
		_ = obj.MarshalPath(benchDeepPath, src)
	}
}
