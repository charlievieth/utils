package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func BenchmarkColorGamutType_UnmarshalJSON(b *testing.B) {
	b.Run("Valid", func(b *testing.B) {
		valid, err := json.Marshal(ColorGamutGeneration1)
		if err != nil {
			b.Fatal(err)
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var c ColorGamutType
			c.UnmarshalJSON(valid)
		}
	})
	b.Run("Invalid", func(b *testing.B) {
		invalid, err := json.Marshal("foobar")
		if err != nil {
			b.Fatal(err)
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var c ColorGamutType
			c.UnmarshalJSON(invalid)
		}
	})
}

func TestColorGamut_JSON(t *testing.T) {
	orig := ColorGamut{
		Red:   XY{0, 1},
		Green: XY{2, 3},
		Blue:  XY{4, 5},
	}
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatal(err)
	}
	var c ColorGamut
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	if c != orig {
		t.Errorf("ColorGamut: want: %+v got: %+v", orig, c)
	}
}

func TestStateUpdateResponse(t *testing.T) {
	tests := []struct {
		Raw string
		Exp StateUpdateResponseBody
	}{
		{
			Raw: `[{"success":{"/lights/1/state/on":true}}]`,
			Exp: StateUpdateResponseBody{
				Success: map[string]interface{}{"/lights/1/state/on": true},
			},
		},
		{
			Raw: `[{"error":{"type":2,"address":"/lights/1/state","description":"body contains invalid json"}}]`,
			Exp: StateUpdateResponseBody{
				Error: &ErrorResponse{
					Type:        2,
					Address:     "/lights/1/state",
					Description: "body contains invalid json",
				},
			},
		},
		{
			Raw: `[{"error":{"type":7,"address":"/lights/1/state/on","description":"invalid value,  foobar }, for parameter, on"}}]`,
			Exp: StateUpdateResponseBody{
				Error: &ErrorResponse{
					Type:        7,
					Address:     "/lights/1/state/on",
					Description: "invalid value,  foobar }, for parameter, on",
				},
			},
		},
	}

	for i, x := range tests {
		var res StateUpdateResponse
		if err := json.Unmarshal([]byte(x.Raw), &res); err != nil {
			t.Errorf("%d (%+v): %s", i, x, err)
			continue
		}
		if len(res) != 1 {
			t.Errorf("%d (%+v): expected 1 response got: %d", i, x, len(res))
		}
		if !reflect.DeepEqual(res[0], x.Exp) {
			t.Errorf("%d (%+v): got: %+v want: %+v", i, x, res[0], x.Exp)
		}
	}
}
