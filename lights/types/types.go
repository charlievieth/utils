package types

// Hue limits values to Max - 1
const (
	MaxUint8  = 254
	MaxUint16 = 65534
)

// String returns a pointer to the string value passed in.
func String(v string) *string {
	return &v
}

// StringValue returns the value of the string pointer passed in or
// "" if the pointer is nil.
func StringValue(v *string) string {
	if v != nil {
		return *v
	}
	return ""
}

// Bool returns a pointer to the bool value passed in.
func Bool(v bool) *bool {
	return &v
}

// BoolValue returns the value of the bool pointer passed in or
// false if the pointer is nil.
func BoolValue(v *bool) bool {
	if v != nil {
		return *v
	}
	return false
}

// Int16 returns a pointer to the int16 value passed in.
func Int16(v int16) *int16 {
	return &v
}

// Int16Value returns the value of the int16 pointer passed in or
// 0 if the pointer is nil.
func Int16Value(v *int16) int16 {
	if v != nil {
		return *v
	}
	return 0
}

// Int32 returns a pointer to the int32 value passed in.
func Int32(v int32) *int32 {
	return &v
}

// Int32Value returns the value of the int32 pointer passed in or
// 0 if the pointer is nil.
func Int32Value(v *int32) int32 {
	if v != nil {
		return *v
	}
	return 0
}

// Uint8 returns a pointer to the uint8 value passed in.
func Uint8(v uint8) *uint8 {
	return &v
}

// Uint8Value returns the value of the uint8 pointer passed in or
// 0 if the pointer is nil.
func Uint8Value(v *uint8) uint8 {
	if v != nil {
		return *v
	}
	return 0
}

func ClampUint8(i int) uint8 {
	if i < 0 {
		return 0
	}
	if i > MaxUint8 {
		return MaxUint8
	}
	return uint8(i)
}

func AddUint8(a, b uint8) uint8 {
	n := uint(a) + uint(b)
	if n > MaxUint8 {
		return MaxUint8
	}
	return uint8(n)
}

func SubUint8(a, b uint8) uint8 {
	if a <= b {
		return 0
	}
	return a - b
}

func ClampUint16(i int) uint16 {
	if i < 0 {
		return 0
	}
	if i > MaxUint16 {
		return MaxUint16
	}
	return uint16(i)
}

func AddUint16(a, b uint16) uint16 {
	n := uint(a) + uint(b)
	if n > MaxUint16 {
		return MaxUint16
	}
	return uint16(n)
}

func SubUint16(a, b uint16) uint16 {
	if a <= b {
		return 0
	}
	return a - b
}

// Uint16 returns a pointer to the uint16 value passed in.
func Uint16(v uint16) *uint16 {
	return &v
}

// Uint16Value returns the value of the uint16 pointer passed in or
// 0 if the pointer is nil.
func Uint16Value(v *uint16) uint16 {
	if v != nil {
		return *v
	}
	return 0
}

/*
// SecondsTimeValue converts an int64 pointer to a time.Time value
// representing seconds since Epoch or time.Time{} if the pointer is nil.
func SecondsTimeValue(v *int64) time.Time {
	if v != nil {
		return time.Unix((*v / 1000), 0)
	}
	return time.Time{}
}

// MillisecondsTimeValue converts an int64 pointer to a time.Time value
// representing milliseconds sinch Epoch or time.Time{} if the pointer is nil.
func MillisecondsTimeValue(v *int64) time.Time {
	if v != nil {
		return time.Unix(0, (*v * 1000000))
	}
	return time.Time{}
}

// TimeUnixMilli returns a Unix timestamp in milliseconds from "January 1, 1970 UTC".
// The result is undefined if the Unix time cannot be represented by an int64.
// Which includes calling TimeUnixMilli on a zero Time is undefined.
//
// This utility is useful for service API's such as CloudWatch Logs which require
// their unix time values to be in milliseconds.
//
// See Go stdlib https://golang.org/pkg/time/#Time.UnixNano for more information.
func TimeUnixMilli(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond/time.Nanosecond)
}
*/
