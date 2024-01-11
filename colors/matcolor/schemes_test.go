package matcolor

import (
	"fmt"
	"image/color"
	"testing"
)

func TestNewSchemes(t *testing.T) {
	k := &Key{
		Primary:        color.RGBA{52, 61, 235, 255},
		Secondary:      color.RGBA{123, 135, 122, 255},
		Tertiary:       color.RGBA{106, 196, 178, 255},
		Error:          color.RGBA{219, 46, 37, 255},
		Neutral:        color.RGBA{133, 131, 121, 255},
		NeutralVariant: color.RGBA{107, 106, 101, 255},
	}
	p := NewPalette(k)
	s := NewSchemes(p)
	fmt.Println(s)
}

func TestNewSchemesFromPrimary(t *testing.T) {
	k := KeyFromPrimary(color.RGBA{0, 0, 255, 255})
	p := NewPalette(k)
	s := NewSchemes(p)
	fmt.Println(s)
}
