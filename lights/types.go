package main

import (
	"encoding/json"
	"errors"
)

type ColorGamutType string

const (
	ColorGamutLivingColors ColorGamutType = "A"
	ColorGamutGeneration1  ColorGamutType = "B"
	ColorGamutFull         ColorGamutType = "C"
	ColorGamutOther        ColorGamutType = "other"
)

func (c ColorGamutType) String() string { return string(c) }

func (c ColorGamutType) Type() string {
	switch c {
	case ColorGamutLivingColors:
		return "Living colors & lightstrip v1 gamut"
	case ColorGamutGeneration1:
		return "Hue generation 1 gamut"
	case ColorGamutFull:
		return "Hue full colors gamut"
	case ColorGamutOther:
		return "Other/not properly defined gamuts"
	default:
		return "Invalid Color Gamut: " + string(c)
	}
}

func unquote(b []byte) []byte {
	if len(b) < 2 || b[0] != '"' || b[len(b)-1] != '"' {
		return b
	}
	return b[1 : len(b)-1]
}

// CEV: consider removing since this will break if
// newer API versions introduce new values.
func (c *ColorGamutType) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	s := unquote(b)
	switch ColorGamutType(s) {
	case ColorGamutLivingColors:
		*c = ColorGamutLivingColors
		return nil
	case ColorGamutGeneration1:
		*c = ColorGamutGeneration1
		return nil
	case ColorGamutFull:
		*c = ColorGamutFull
		return nil
	default:
		*c = ColorGamutOther
		if len(b) > 32 {
			b = append(b[:32-len("...")], "..."...)
		}
		return errors.New("lights: invalid ColorGamutType: " + string(b))
	}
}

type ColorMode string

const (
	ColorModeHueSaturation ColorMode = "hs"
	ColorModeXY            ColorMode = "xy"
	ColorModeTemperature   ColorMode = "ct"
	ColorModeOther         ColorMode = "other"
)

func (c ColorMode) String() string { return string(c) }

// CEV: consider removing since this will break if
// newer API versions introduce new values.
func (c *ColorMode) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	s := unquote(b)
	switch ColorMode(s) {
	case ColorModeHueSaturation:
		*c = ColorModeHueSaturation
		return nil
	case ColorModeXY:
		*c = ColorModeXY
		return nil
	case ColorModeTemperature:
		*c = ColorModeTemperature
		return nil
	default:
		*c = ColorModeOther
		if len(b) > 32 {
			b = append(b[:32-len("...")], "..."...)
		}
		return errors.New("lights: invalid ColorMode: " + string(b))
	}
}

type ColorGamut struct {
	Red   XY `json:"red"`   // max X, max Y
	Green XY `json:"green"` // max X, max Y
	Blue  XY `json:"blue"`  // max X, max Y
}

// CEV: consider removing since this will break if
// newer API versions introduce new values.
func (c *ColorGamut) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var gamut [3][2]float32
	if err := json.Unmarshal(b, &gamut); err != nil {
		return err
	}
	*c = ColorGamut{
		Red:   XY{X: gamut[0][0], Y: gamut[0][1]},
		Green: XY{X: gamut[1][0], Y: gamut[1][1]},
		Blue:  XY{X: gamut[2][0], Y: gamut[2][1]},
	}
	return nil
}

func (c ColorGamut) MarshalJSON() ([]byte, error) {
	gamut := [3][2]float32{
		0: {c.Red.X, c.Red.Y},
		1: {c.Green.X, c.Green.Y},
		2: {c.Blue.X, c.Blue.Y},
	}
	return json.Marshal(&gamut)
}

type LightState struct {
	On               bool      `json:"on"`
	Reachable        *bool     `json:"reachable,omitempty"`
	Brightness       uint8     `json:"bri"`
	Saturation       *uint8    `json:"sat,omitempty"`
	Hue              *uint16   `json:"hue,omitempty"`
	ColorTemperature uint16    `json:"ct"`
	Alert            string    `json:"alert"`
	Effect           string    `json:"effect,omitempty"`
	ColorMode        ColorMode `json:"colormode"`
	XY               *XY       `json:"xy,omitempty"`
}

type ErrorResponse struct {
	Type        int    `json:"type"`
	Address     string `json:"address"`
	Description string `json:"description"`
}

////////////////////////////////////////////////////////////////////////////////
//
// TODO: use this
//
type Response struct {
	// Address => response
	Success map[string]interface{}

	// Address => error
	Errors map[string]ErrorResponse
}

// TODO: simplify
//
// TODO: rename to PutResponse or something
type StateUpdateResponse []StateUpdateResponseBody

func (s StateUpdateResponse) Response() *Response {
	var (
		success map[string]interface{}
		errors  map[string]ErrorResponse
	)
	for _, r := range s {
		for addr, val := range r.Success {
			if success == nil {
				success = make(map[string]interface{})
			}
			success[addr] = val
		}
		if e := r.Error; e != nil {
			if errors == nil {
				errors = make(map[string]ErrorResponse)
			}
			errors[e.Address] = *e
		}
	}
	return &Response{
		Success: success,
		Errors:  errors,
	}
}

// CEV: rename
type StateUpdateResponseBody struct {
	// address => value
	Success map[string]interface{} `json:"success,omitempty"`
	Error   *ErrorResponse         `json:"error,omitempty"`
}

// [{"success":{"/lights/1/state/on":true}}]
// [{"error":{"type":2,"address":"/lights/1/state","description":"body contains invalid json"}}]
// [{"error":{"type":7,"address":"/lights/1/state/on","description":"invalid value,  foobar }, for parameter, on"}}]
// [{"error":{"type":7,"address":"/lights/1/state/on","description":"invalid value, 1,, for parameter, on"}},{"error":{"type":7,"address":"/lights/1/state/bri","description":"invalid value, true}, for parameter, bri"}}]
// [
//   {
//     "error": {
//       "type": 7,
//       "address": "/lights/1/state/on",
//       "description": "invalid value, 1,, for parameter, on"
//     }
//   },
//   {
//     "error": {
//       "type": 7,
//       "address": "/lights/1/state/bri",
//       "description": "invalid value, true}, for parameter, bri"
//     }
//   }
// ]

type LightStateRequest struct {
	// On/Off state of the light. On=true, Off=false
	On *bool `json:"on,omitempty"`

	// The brightness value to set the light to.Brightness is a scale from 1
	// (the minimum the light is capable of) to 254 (the maximum).
	Brightness *uint8 `json:"bri,omitempty"`

	// Saturation of the light. 254 is the most saturated (colored) and 0 is
	// the least saturated (white).
	Saturation *uint8 `json:"sat,omitempty"`

	// The hue value to set light to.The hue value is a wrapping value between
	// 0 and 65535. Both 0 and 65535 are red, 25500 is green and 46920 is blue.
	Hue *uint16 `json:"hue,omitempty"`

	// The Mired color temperature of the light. 2012 connected lights are
	// capable of 153 (6500K) to 500 (2000K).
	ColorTemperature *uint16 `json:"ct,omitempty"`

	// The alert effect,is a temporary change to the bulb’s state, and has one
	// of the following values:
	//
	// * "none" – The light is not performing an alert effect.
	//
	// * "select" – The light is performing one breathe cycle.
	//
	// * "lselect" – The light is performing breathe cycles for 15 seconds or
	//   until an "alert": "none" command is received.Note that this contains
	//   the last alert sent to the light and not its current state. i.e. After
	//   the breathe cycle has finished the bridge does not reset the alert to
	//   “none“.
	//
	Alert *string `json:"alert,omitempty"`

	// The dynamic effect of the light. Currently “none” and “colorloop” are
	// supported. Other values will generate an error of type 7.Setting the
	// effect to colorloop will cycle through all hues using the current
	// brightness and saturation settings.
	Effect *string `json:"effect,omitempty"`

	// The x and y coordinates of a color in CIE color space.The first entry is
	// the x coordinate and the second entry is the y coordinate. Both x and y
	// must be between 0 and 1.
	//
	// If the specified coordinates are not in the CIE color space, the closest
	// color to the coordinates will be chosen.
	XY *XY `json:"xy,omitempty"`

	// The duration of the transition from the light’s current state to the new
	// state. This is given as a multiple of 100ms and defaults to 4 (400ms).
	// For example, setting transitiontime:10 will make the transition last 1
	TransitionTime *uint16 `json:"transitiontime,omitempty"`

	// Increments or decrements the value of the brightness.  bri_inc is
	// ignored if the bri attribute is provided. Any ongoing bri transition is
	// stopped. Setting a value of 0 also stops any ongoing transition. The
	// bridge will return the bri value after the increment is performed.
	BrightnessInc *int16 `json:"bri_inc,omitempty"`

	// Increments or decrements the value of the sat.  sat_inc is ignored if
	// the sat attribute is provided. Any ongoing sat transition is stopped.
	// Setting a value of 0 also stops any ongoing transition. The bridge will
	// return the sat value after the increment is performed.
	SaturationInc *int16 `json:"sat_inc,omitempty"`

	// Increments or decrements the value of the hue.   hue_inc is ignored if
	// the hue attribute is provided. Any ongoing color transition is stopped.
	// Setting a value of 0 also stops any ongoing transition. The bridge will
	// return the hue value after the increment is performed.Note if the
	// resulting values are < 0 or > 65535 the result is wrapped.
	HueInc *int32 `json:"hue_inc,omitempty"`

	// Increments or decrements the value of the ct. ct_inc is ignored if the
	// ct attribute is provided. Any ongoing color transition is stopped.
	// Setting a value of 0 also stops any ongoing transition. The bridge will
	// return the ct value after the increment is performed.
	ColorTemperatureInc *int32 `json:"ct_inc,omitempty"`

	// Increments or decrements the value of the xy.  xy_inc is ignored if the
	// xy attribute is provided. Any ongoing color transition is stopped.
	// Setting a value of 0 also stops any ongoing transition. Will stop at
	// it’s gamut boundaries. The bridge will return the xy value after the
	// increment is performed. Max value [0.5, 0.5].
	XYInc *XY `json:"xy_inc,omitempty"`

	// The scene identifier if the scene you wish to recall.
	//
	// Light groups only.
	Scene *string `json:"scene,omitempty"`
}

type SoftwareUpdate struct {
	LastInstall PhueTime `json:"lastinstall"`
	State       string   `json:"state"`
}

type Streaming struct {
	Renderer bool `json:"renderer"`
	Proxy    bool `json:"proxy"`
}

type MinMax struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type Control struct {
	MinDimLevel      int             `json:"mindimlevel"`
	MaxLumen         int             `json:"maxlumen"`
	ColorTemperature MinMax          `json:"ct"`
	ColorGamutType   *ColorGamutType `json:"colorgamuttype,omitempty"`
	ColorGamut       *ColorGamut     `json:"colorgamut,omitempty"`
}

type Capabilities struct {
	Certified bool      `json:"certified"`
	Control   Control   `json:"control"`
	Streaming Streaming `json:"streaming"`
}

// TODO: might wanna add "omitempty" tag
// TODO: add ID field
type Light struct {
	ID int `json:"-"`

	Type              string         `json:"type"`
	Name              string         `json:"name"`
	ProductName       string         `json:"productname"`
	ManufacturerName  string         `json:"manufacturername"`
	ModelID           string         `json:"modelid"`
	UniqueID          string         `json:"uniqueid"`
	LuminaireUniqueID string         `json:"luminaireuniqueid"`
	SoftwareVersion   string         `json:"swversion"`
	Streaming         Streaming      `json:"streaming"`
	Capabilities      Capabilities   `json:"capabilities"`
	Config            LightConfig    `json:"config"`
	State             LightState     `json:"state"`
	SoftwareUpdate    SoftwareUpdate `json:"swupdate"`
}

type LightConfig struct {
	ArcheType string `json:"archetype"`
	Function  string `json:"function"`
	Direction string `json:"direction"`
}

type Lights map[string]Light

type GroupType string

const (
	// A special group containing all lights in the system, and is not returned
	// by the ‘get all groups’ command. This group is not visible, and cannot
	// be created, modified or deleted using the API.
	//
	// API Version: 1.0
	GroupType0 GroupType = "0"

	// Multisource luminaire group A lighting installation of default groupings
	// of hue lights. The bridge will pre-install these groups for ease of use.
	// This type cannot be created manually.  Also, a light can only be in a
	// maximum of one luminaire group. See multisource luminaires for more
	// info.
	//
	// API Version: 1.4
	GroupTypeLuminaire GroupType = "Luminaire"

	// LightSource group A group of lights which is created by the bridge based
	// on multisource luminaire attributes of Zigbee light resource.
	//
	// API Version: 1.4
	GroupTypeLightSource GroupType = "Lightsource"

	// LightGroup group A group of lights that can be controlled together. This
	// the default group type that the bridge generates for user created
	// groups. Default type when no type is given on creation.
	//
	// API Version: 1.4
	GroupTypeLightGroup GroupType = "LightGroup"

	// Room A group of lights that are physically located in the same place in
	// the house. Rooms behave similar as light groups, except: (1) A room can
	// be empty and contain 0 lights, (2) a light is only allowed in one room
	// and (3) a room isn’t automatically deleted when all lights in that room
	// are deleted.
	//
	// API Version: 1.11
	GroupTypeRoom GroupType = "Room"

	// Zones describe a group of lights that can be controlled together. Zones
	// can be empty and contain 0 lights. A light is allowed to be in multiple
	// zones.
	//
	// API Version: 1.30
	GroupTypeZone GroupType = "Zone"

	// Represents an entertainment setup.
	//
	// Entertainment group describe a group of lights that are used in an
	// entertainment setup. Locations describe the relative position of the
	// lights in an entertainment setup. E.g. for TV the position is relative
	// to the TV. Can be used to configure streaming sessions.
	//
	// Entertainment group behave in a similar way as light groups, with the
	// exception: it can be empty and contain 0 lights. The group is also not
	// automatically recycled when lights are deleted. The group of lights can
	// be controlled together as in LightGroup.
	//
	// API Version: 1.22
	GroupTypeEntertainment GroupType = "Entertainment"
)

var knownGroupTypes = map[GroupType]struct{}{
	GroupType0:             {},
	GroupTypeLuminaire:     {},
	GroupTypeLightSource:   {},
	GroupTypeLightGroup:    {},
	GroupTypeRoom:          {},
	GroupTypeZone:          {},
	GroupTypeEntertainment: {},
}

func (t GroupType) Valid() bool {
	_, ok := knownGroupTypes[t]
	return ok
}

func (t GroupType) String() string { return string(t) }

// TODO: consider renaming GroupClass
type RoomClass string

const (
	RoomClassLivingRoom  RoomClass = "Living room"
	RoomClassKitchen     RoomClass = "Kitchen"
	RoomClassDining      RoomClass = "Dining"
	RoomClassBedroom     RoomClass = "Bedroom"
	RoomClassKidsBedroom RoomClass = "Kids bedroom"
	RoomClassBathroom    RoomClass = "Bathroom"
	RoomClassNursery     RoomClass = "Nursery"
	RoomClassRecreation  RoomClass = "Recreation"
	RoomClassOffice      RoomClass = "Office"
	RoomClassGym         RoomClass = "Gym"
	RoomClassHallway     RoomClass = "Hallway"
	RoomClassToilet      RoomClass = "Toilet"
	RoomClassFrontDoor   RoomClass = "Front door"
	RoomClassGarage      RoomClass = "Garage"
	RoomClassTerrace     RoomClass = "Terrace"
	RoomClassGarden      RoomClass = "Garden"
	RoomClassDriveway    RoomClass = "Driveway"
	RoomClassCarport     RoomClass = "Carport"
	RoomClassOther       RoomClass = "Other"

	// API Version 1.30
	RoomClassHome        RoomClass = "Home"
	RoomClassDownstairs  RoomClass = "Downstairs"
	RoomClassUpstairs    RoomClass = "Upstairs"
	RoomClassTopFloor    RoomClass = "Top floor"
	RoomClassAttic       RoomClass = "Attic"
	RoomClassGuestRoom   RoomClass = "Guest room"
	RoomClassStaircase   RoomClass = "Staircase"
	RoomClassLounge      RoomClass = "Lounge"
	RoomClassManCave     RoomClass = "Man cave"
	RoomClassComputer    RoomClass = "Computer"
	RoomClassStudio      RoomClass = "Studio"
	RoomClassMusic       RoomClass = "Music"
	RoomClassTV          RoomClass = "TV"
	RoomClassReading     RoomClass = "Reading"
	RoomClassCloset      RoomClass = "Closet"
	RoomClassStorage     RoomClass = "Storage"
	RoomClassLaundryRoom RoomClass = "Laundry room"
	RoomClassBalcony     RoomClass = "Balcony"
	RoomClassPorch       RoomClass = "Porch"
	RoomClassBarbecue    RoomClass = "Barbecue"
	RoomClassPool        RoomClass = "Pool"
)

var knownRoomClasses = map[RoomClass]struct{}{
	RoomClassLivingRoom:  {},
	RoomClassKitchen:     {},
	RoomClassDining:      {},
	RoomClassBedroom:     {},
	RoomClassKidsBedroom: {},
	RoomClassBathroom:    {},
	RoomClassNursery:     {},
	RoomClassRecreation:  {},
	RoomClassOffice:      {},
	RoomClassGym:         {},
	RoomClassHallway:     {},
	RoomClassToilet:      {},
	RoomClassFrontDoor:   {},
	RoomClassGarage:      {},
	RoomClassTerrace:     {},
	RoomClassGarden:      {},
	RoomClassDriveway:    {},
	RoomClassCarport:     {},
	RoomClassOther:       {},

	// API Version 1.30
	RoomClassHome:        {},
	RoomClassDownstairs:  {},
	RoomClassUpstairs:    {},
	RoomClassTopFloor:    {},
	RoomClassAttic:       {},
	RoomClassGuestRoom:   {},
	RoomClassStaircase:   {},
	RoomClassLounge:      {},
	RoomClassManCave:     {},
	RoomClassComputer:    {},
	RoomClassStudio:      {},
	RoomClassMusic:       {},
	RoomClassTV:          {},
	RoomClassReading:     {},
	RoomClassCloset:      {},
	RoomClassStorage:     {},
	RoomClassLaundryRoom: {},
	RoomClassBalcony:     {},
	RoomClassPorch:       {},
	RoomClassBarbecue:    {},
	RoomClassPool:        {},
}

func (r RoomClass) Valid() bool {
	_, ok := knownRoomClasses[r]
	return ok
}

func (r RoomClass) String() string { return string(r) }

type GroupState struct {
	AllOn bool `json:"all_on"`
	AnyOn bool `json:"any_on"`
}

type GroupPresence struct {
	// TODO: figure out whats in the State object
	State json.RawMessage `json:"state"`

	// Last time the combined state was changed
	//
	// TODO: figure out the time format so that we can parse it
	LastUpdated string `json:"lastupdated"`

	// Any sensor (i.e one or more) in the group detected presence
	Presence bool `json:"presence"`

	// All sensors in the group detected presence
	PresenceAll bool `json:"presence_all"`
}

type GroupLightLevel struct {
	// TODO: figure out whats in the State object
	State json.RawMessage `json:"state"`

	// Last time the combined state was updated
	//
	// TODO: figure out the time format so that we can parse it
	LastUpdated string `json:"lastupdated"`

	// There is not sufficient light in the group (for at least one sensor)
	Dark bool `json:"dark"`

	// All sensors do not detect sufficient light
	DarkAll bool `json:"dark_all"`

	// There is sufficient light in the group (for all sensors)
	Daylight bool `json:"daylight"`

	// Some sensors detect there is sufficient light
	DaylightAny bool `json:"daylight_any"`

	// Average light level in the group
	LightLevel int `json:"lightlevel"`

	// Minimum measured light level
	LightLevelMin int `json:"lightlevel_min"`

	// Maximum measured light level
	LightLevelMax int `json:"lightlevel_max"`
}

type Group struct {
	// Group ID
	ID int `json:"-"`

	// Human readable name of the group. If name is not specified one is
	// generated for you (default name is “Group”)
	Name string `json:"name"`

	// The ordered set of light ids from the lights which are  in the group.
	// This resource shall contain an array of at least one element with the
	// exception of the “Room” type: The Room type may contain an empty lights
	// array. Each element can appear only once. Order of lights on creation is
	// preserved. A light id must be an existing light resource in /lights. If
	// an invalid lights resource is given, error 7 shall be returned and the
	// group is not created. There shall be no change in the lights.
	//
	// Light id can be null if a group has been automatically create by the
	// bridge and a light source is not yet available
	Lights []string `json:"lights"`

	// 	The ordered set of sensor ids from the sensors which are in the group.
	// 	The array can be empty.
	//
	// A sensor id must be an existing sensor resource in /sensors. If an
	// invalid sensor resource is given, error 7 shall be returned and the
	// group is not created.
	Sensors []string `json:"sensors"`

	// Type of the Group. If not provided on creation a “LightGroup” is
	// created. Supported types:
	Type GroupType `json:"type"`

	// Contains a state representation of the group
	State GroupState `json:"state"`

	// When true: Resource is automatically deleted when not referenced anymore
	// in any resource link. Only on creation of resource. “false” when
	// omitted.
	Recycle bool `json:"recycle"`

	// The room class of the light.
	Class RoomClass `json:"class"`

	// Is used to execute actions on all lights in a group.
	Action LightState `json:"action"`

	// NB: the definition of these types are incomplete

	// Only exists if sensors array contains a presence sensor of type
	// “ZLLPresence”, “CLIPPresence” or “Geofence”. This object contains a
	// state object which contains the aggregated state of the sensors
	Presence *GroupPresence `json:"presence,omitempty"`

	// Only exists if sensors array contains a light sensor of type
	// “ZLLLightlevel” or ”CLIPLightLevel”. This object contains a state object
	// which contains the aggregated state of the sensors
	LightLevel *GroupLightLevel `json:"lightlevel,omitempty"`
}

/*
func (s *StateUpdateRequest) GetOn() bool {
	if s != nil && s.On != nil {
		return *s.On
	}
	return false
}
func (s *StateUpdateRequest) GetBrightness() uint8 {
	if s != nil && s.Brightness != nil {
		return *s.Brightness
	}
	return 0
}
func (s *StateUpdateRequest) GetSaturation() uint8 {
	if s != nil && s.Saturation != nil {
		return *s.Saturation
	}
	return 0
}
func (s *StateUpdateRequest) GetHue() uint16 {
	if s != nil && s.Hue != nil {
		return *s.Hue
	}
	return 0
}
func (s *StateUpdateRequest) GetColorTemperature() uint16 {
	if s != nil && s.ColorTemperature != nil {
		return *s.ColorTemperature
	}
	return 0
}
func (s *StateUpdateRequest) GetAlert() string {
	if s != nil && s.Alert != nil {
		return *s.Alert
	}
	return ""
}
func (s *StateUpdateRequest) GetEffect() string {
	if s != nil && s.Effect != nil {
		return *s.Effect
	}
	return ""
}
func (s *StateUpdateRequest) GetColorMode() ColorMode {
	if s != nil && s.ColorMode != nil {
		return *s.ColorMode
	}
	return ColorModeOther
}
func (s *StateUpdateRequest) GetXY() XY {
	if s != nil && s.XY != nil {
		return *s.XY
	}
	return XY{}
}
*/
