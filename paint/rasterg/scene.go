package main

import (
	"log"
	"math"

	"github.com/linebender/vello/encoding"
	"github.com/linebender/vello/peniko/kurbo"
)

type Scene struct {
	encoding encoding.Encoding
}

func (s *Scene) Reset() {
	s.encoding.Reset()
}

func (s *Scene) BumpEstimate(transform kurbo.Transform) encoding.BumpAllocatorMemory {
	return s.encoding.Estimator.Tally(transform)
}

func (s *Scene) Encoding() *encoding.Encoding {
	return &s.encoding
}

func (s *Scene) EncodingMut() *encoding.Encoding {
	return &s.encoding
}

func (s *Scene) PushLayer(blend encoding.BlendMode, alpha float32, transform kurbo.Transform, clip kurbo.Shape) {
	if blend.Mix == encoding.MixClip && alpha != 1.0 {
		log.Println("Clip mix mode used with semitransparent alpha")
	}
	t := encoding.TransformFromKurbo(transform)
	s.encoding.EncodeTransform(t)
	s.encoding.EncodeFillStyle(encoding.FillNonZero)
	if !s.encoding.EncodeShape(clip, true) {
		s.encoding.EncodeEmptyShape()
		if s.encoding.Estimator != nil {
			path := []kurbo.PathEl{
				kurbo.PathElMoveTo(kurbo.Point{}),
				kurbo.PathElLineTo(kurbo.Point{}),
			}
			s.encoding.Estimator.CountPath(path, t, nil)
		}
	} else if s.encoding.Estimator != nil {
		s.encoding.Estimator.CountPath(clip.PathElements(0.1), t, nil)
	}
	s.encoding.EncodeBeginClip(blend, math.Clamp(float64(alpha), 0.0, 1.0))
}

func (s *Scene) PopLayer() {
	s.encoding.EncodeEndClip()
}

func (s *Scene) DrawBlurredRoundedRect(transform kurbo.Transform, rect kurbo.Rect, brush encoding.Color, radius, stdDev float64) {
	kernelSize := 2.5 * stdDev
	shape := rect.Inflate(kernelSize, kernelSize)
	s.DrawBlurredRoundedRectIn(shape, transform, rect, brush, radius, stdDev)
}

func (s *Scene) DrawBlurredRoundedRectIn(shape kurbo.Shape, transform kurbo.Transform, rect kurbo.Rect, brush encoding.Color, radius, stdDev float64) {
	t := encoding.TransformFromKurbo(transform)
	s.encoding.EncodeTransform(t)
	s.encoding.EncodeFillStyle(encoding.FillNonZero)
	if s.encoding.EncodeShape(shape, true) {
		brushTransform := encoding.TransformFromKurbo(transform.PreTranslate(rect.Center().ToVec2()))
		if s.encoding.EncodeTransform(brushTransform) {
			s.encoding.SwapLastPathTags()
		}
		s.encoding.EncodeBlurredRoundedRect(brush, float32(rect.Width()), float32(rect.Height()), float32(radius), float32(stdDev))
	}
}

func (s *Scene) Fill(style encoding.Fill, transform kurbo.Transform, brush encoding.BrushRef, brushTransform kurbo.Transform, shape kurbo.Shape) {
	t := encoding.TransformFromKurbo(transform)
	s.encoding.EncodeTransform(t)
	s.encoding.EncodeFillStyle(style)
	if s.encoding.EncodeShape(shape, true) {
		if brushTransform != nil {
			if s.encoding.EncodeTransform(encoding.TransformFromKurbo(transform.Mul(brushTransform))) {
				s.encoding.SwapLastPathTags()
			}
		}
		s.encoding.EncodeBrush(brush, 1.0)
	}
}

func (s *Scene) Stroke(style encoding.Stroke, transform kurbo.Transform, brush encoding.BrushRef, brushTransform kurbo.Transform, shape kurbo.Shape) {
	const shapeTolerance = 0.01
	const strokeTolerance = shapeTolerance

	const gpuStrokes = true
	if gpuStrokes {
		t := encoding.TransformFromKurbo(transform)
		s.encoding.EncodeTransform(t)
		s.encoding.EncodeStrokeStyle(style)

		var encodeResult bool
		if len(style.DashPattern) == 0 {
			encodeResult = s.encoding.EncodeShape(shape, false)
		} else {
			dashed := kurbo.Dash(shape.PathElements(shapeTolerance), style.DashOffset, style.DashPattern)
			encodeResult = s.encoding.EncodePathElements(dashed, false)
		}

		if encodeResult {
			if brushTransform != nil {
				if s.encoding.EncodeTransform(encoding.TransformFromKurbo(transform.Mul(brushTransform))) {
					s.encoding.SwapLastPathTags()
				}
			}
			s.encoding.EncodeBrush(brush, 1.0)
		}
	} else {
		stroked := kurbo.Stroke(shape.PathElements(shapeTolerance), style, kurbo.DefaultDashJoiner, strokeTolerance)
		s.Fill(encoding.FillNonZero, transform, brush, brushTransform, stroked)
	}
}

func (s *Scene) DrawImage(image encoding.Image, transform kurbo.Transform) {
	s.Fill(
		encoding.FillNonZero,
		transform,
		image,
		nil,
		kurbo.Rect{
			Min: kurbo.Point{},
			Max: kurbo.Point{
				X: float64(image.Width),
				Y: float64(image.Height),
			},
		},
	)
}

func (s *Scene) DrawGlyphs(font encoding.Font) *DrawGlyphs {
	return NewDrawGlyphs(s, font)
}

func (s *Scene) Append(other *Scene, transform kurbo.Transform) {
	t := encoding.TransformFromKurbo(transform)
	s.encoding.Append(&other.encoding, t)
}

func (s *Scene) SetBrushAlpha(alpha float32) *Scene {
            s.brushAlpha = alpha
            return s
        }

        // Draw encodes a fill or stroke for the given sequence of glyphs and consumes the builder.
        //
        // The style parameter accepts either Fill or Stroke types.
        //
        // This supports emoji fonts in COLR and bitmap formats.
        // style is ignored for these fonts.
        //
        // For these glyphs, the given brush is used as the "foreground color", and should
        // be Solid for maximum compatibility.
        func (s *Scene) Draw(style StyleRef, glyphs []Glyph) {
            fontIndex := s.run.font.index
            font := skrifa.FontRefFromIndex(s.run.font.data, fontIndex)
            bitmaps := bitmap.NewBitmapStrikes(font)
            if font.Colr() != nil && font.Cpal() != nil || !bitmaps.IsEmpty() {
                s.tryDrawColr(style, glyphs)
            } else {
                // Shortcut path - no need to test each glyph for a colr outline
                outlineCount := s.drawOutlineGlyphs(style, glyphs)
                if outlineCount == 0 {
                    s.encoding.resources.normalizedCoords = s.encoding.resources.normalizedCoords[:s.run.normalizedCoords.start]
                }
            }
        }

        func (s *Scene) drawOutlineGlyphs(style StyleRef, glyphs []Glyph) int {
            resources := &s.encoding.resources
            s.run.style = style
            resources.glyphs = append(resources.glyphs, glyphs...)
            s.run.glyphs.end = len(resources.glyphs)
            if len(s.run.glyphs) == 0 {
                return 0
            }
            index := len(resources.glyphRuns)
            resources.glyphRuns = append(resources.glyphRuns, s.run)
            resources.patches = append(resources.patches, Patch{Type: GlyphRunPatch, Index: index})
            s.encoding.encodeBrush(s.brush, s.brushAlpha)
            // Glyph run resolve step affects transform and style state in a way
            // that is opaque to the current encoding.
            // See <https://github.com/linebender/vello/issues/424>
            s.encoding.forceNextTransformAndStyle()
            return len(s.run.glyphs)
        }

        func (s *Scene) tryDrawColr(style StyleRef, glyphs []Glyph) {
            fontIndex := s.run.font.index
            blob := s.run.font.data
            font := skrifa.FontRefFromIndex(blob, fontIndex)
            upem := float32(font.Head().UnitsPerEm())
            runTransform := s.run.transform.ToKurbo()
            colrScale := affine.ScaleNonUniform(
                float64(s.run.fontSize/upem),
                float64(-s.run.fontSize/upem),
            )

            colorCollection := font.ColorGlyphs()
            bitmaps := bitmap.NewBitmapStrikes(font)
            var finalGlyph *Glyph
            outlineCount := 0
            coords := s.encoding.resources.normalizedCoords[s.run.normalizedCoords.start:]
            location := NewLocationRef(coords)
            for {
                ppem := s.run.fontSize
                outlineGlyphs := make([]Glyph, 0)
                for _, glyph := range glyphs {
                    glyphID := GlyphID(glyph.ID)
                    if color := colorCollection.Get(glyphID); color != nil {
                        finalGlyph = &Glyph{EmojiLikeGlyph: &ColorGlyph{Color: color}, Glyph: glyph}
                        break
                    } else if bitmap := bitmaps.GlyphForSize(bitmap.Size{PPem: ppem}, glyphID); bitmap != nil {
                        finalGlyph = &Glyph{EmojiLikeGlyph: &BitmapGlyph{Bitmap: bitmap}, Glyph: glyph}
                        break
                    } else {
                        outlineGlyphs = append(outlineGlyphs, glyph)
                    }
                }
                s.run.glyphs.start = s.run.glyphs.end
                s.run.streamOffsets = s.encoding.streamOffsets()
                outlineCount += s.drawOutlineGlyphs(style, outlineGlyphs)

                if finalGlyph == nil {
                    // All of the remaining glyphs were outline glyphs
                    break
                }

                switch emoji := finalGlyph.EmojiLikeGlyph.(type) {
                // TODO: This really needs to be moved to resolve time to get proper caching, etc.
                case *BitmapGlyph:
                    // implementation
                case *ColorGlyph:
                    // implementation
                }
