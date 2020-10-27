package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
)

type XY struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

func (x *XY) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var xy [2]float32
	if err := json.Unmarshal(b, &xy); err != nil {
		return err
	}
	*x = XY{X: xy[0], Y: xy[1]}
	return nil
}

func (x XY) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]float32{x.X, x.Y})
}

func XYToColor(x, y float64) color.RGBA {
	const Y = 1.0
	z := 1.0 - x - y // TODO: move to Z
	X := (Y / y) * x
	Z := (Y / y) * z

	// Convert to RGB using Wide RGB D65 conversion
	r := X*1.656492 - Y*0.354851 - Z*0.255038
	g := -X*0.707196 + Y*1.655397 + Z*0.036152
	b := X*0.051713 - Y*0.121364 + Z*1.011530

	switch {
	case r > b && r > g && r > 1.0:
		// red is too big
		g /= r
		b /= r
		r = 1.0
	case g > b && g > r && g > 1.0:
		// green is too big
		r /= g
		b /= g
		g = 1.0
	case b > r && b > g && b > 1.0:
		// blue is too big
		r /= b
		g /= b
		b = 1.0
	}

	// TODO: make sure this is correct
	//
	// Apply reverse gamma correction
	if r <= 0.0031308 {
		r = 12.92 * r
	} else {
		r = (1.0+0.055)*math.Pow(r, (1.0/2.4)) - 0.055
	}
	if g <= 0.0031308 {
		g = 12.92 * g
	} else {
		g = (1.0+0.055)*math.Pow(g, (1.0/2.4)) - 0.055
	}
	if b <= 0.0031308 {
		b = 12.92 * b
	} else {
		b = (1.0+0.055)*math.Pow(b, (1.0/2.4)) - 0.055
	}

	switch {
	case r > b && r > g:
		// red is biggest
		if r > 1.0 {
			g /= r
			b /= r
			r = 1.0
		}
	case g > b && g > r:
		// green is biggest
		if g > 1.0 {
			r /= g
			b /= g
			g = 1.0
		}
	case b > r && b > g:
		// blue is biggest
		if b > 1.0 {
			r /= b
			g /= b
			b = 1.0
		}
	}

	fmt.Println("R:", r, uint8(r*math.MaxUint8))
	fmt.Println("G:", g, uint8(g*math.MaxUint8))
	fmt.Println("B:", b, uint8(b*math.MaxUint8))

	return color.RGBA{
		R: uint8(r * math.MaxUint8),
		G: uint8(g * math.MaxUint8),
		B: uint8(b * math.MaxUint8),
	}
}

func (x XY) RGB(brightness uint8) color.RGBA {
	const MaxBrightness = 254

	// https://developers.meethue.com/develop/application-design-guidance/color-conversion-formulas-rgb-to-xy-and-back/
	//
	// Calculate XYZ values
	xX := float64(x.X)
	xY := float64(x.Y)
	z := 1.0 - xX - xY
	Y := float64(brightness) / MaxBrightness
	fmt.Printf("X: %f Y: %f\n", x.X, x.Y)     // WARN
	fmt.Println("BRIGHTNESS:", Y, brightness) // WARN
	Y = 0.75
	X := (Y / xY) * xX
	Z := (Y / xY) * z

	// Convert to RGB using Wide RGB D65 conversion
	r := X*1.656492 - Y*0.354851 - Z*0.255038
	g := -X*0.707196 + Y*1.655397 + Z*0.036152
	b := X*0.051713 - Y*0.121364 + Z*1.011530

	// WARN: something here is broken the follow code fails
	// to bound R to 0..1
	//
	// Apply reverse gamma correction
	if r <= 0.0031308 {
		r = 12.92 * r
	} else {
		r = (1.0+0.055)*math.Pow(r, (1.0/2.4)) - 0.055
	}
	if g <= 0.0031308 {
		g = 12.92 * g
	} else {
		g = (1.0+0.055)*math.Pow(g, (1.0/2.4)) - 0.055
	}
	if b <= 0.0031308 {
		b = 12.92 * b
	} else {
		b = (1.0+0.055)*math.Pow(b, (1.0/2.4)) - 0.055
	}
	fmt.Printf("XY: R: %f G: %f B: %f\n", r, g, b) // WARN

	// WARN: we should not need this!!!
	clamp := func(f float64) uint8 {
		if f >= 1 {
			return MaxBrightness
		}
		if f <= 0 {
			return 0
		}
		return uint8(f * MaxBrightness)
	}
	return color.RGBA{
		R: clamp(r),
		G: clamp(g),
		B: clamp(b),
	}
}
