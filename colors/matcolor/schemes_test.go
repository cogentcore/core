package matcolor

import (
	"fmt"
	"image/color"
	"testing"
)

func TestNewSchemes(t *testing.T) {
	k := &Key{
		Primary:        color.RGBA{R: 52, G: 61, B: 235, A: 255},
		Secondary:      color.RGBA{R: 123, G: 135, B: 122, A: 255},
		Tertiary:       color.RGBA{R: 106, G: 196, B: 178, A: 255},
		Error:          color.RGBA{R: 219, G: 46, B: 37, A: 255},
		Neutral:        color.RGBA{R: 133, G: 131, B: 121, A: 255},
		NeutralVariant: color.RGBA{R: 107, G: 106, B: 101, A: 255},
	}
	p := NewPalette(k)
	s := NewSchemes(p)
	fmt.Println(s)
}

func TestNewSchemesFromPrimary(t *testing.T) {
	k := KeyFromPrimary(color.RGBA{B: 255, A: 255})
	p := NewPalette(k)
	s := NewSchemes(p)
	fmt.Println(s)
}
