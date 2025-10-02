// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package pdf

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
)

func TestPDFPath(t *testing.T) {
	p := ppath.MustParseSVGPath("L20 0")
	var b bytes.Buffer
	pd := New(&b, 50, 50)

	sty := styles.NewPaint()
	sty.Defaults()
	sty.Stroke.Color = colors.Uniform(colors.Blue)
	sty.Fill.Color = colors.Uniform(colors.Red)
	sty.Stroke.Width.Px(2)
	sty.ToDots()

	pd.RenderPath(p, sty, math32.Translate2D(20, 20))
	pd.Close()

	fmt.Println(b.String())
	os.Mkdir("testdata", 0777)
	os.WriteFile("testdata/path.pdf", b.Bytes(), 0666)

	//	pdfCompress = false
	//	buf := &bytes.Buffer{}
	//	c.WritePDF(buf)
	//	test.T(t, buf.String(), `%PDF-1.7
	//1 0 obj
	//<< /Length 14 >> stream
	//0 0 m 10 0 l f
	//endstream
	//endobj
	//2 0 obj
	//<< /Type /Page /Contents 1 0 R /Group << /Type /Group /CS /DeviceRGB /I true /S /Transparency >> /MediaBox [0 0 10 10] /Parent 2 0 R /Resources << >> >>
	//endobj
	//3 0 obj
	//<< /Type /Pages /Count 1 /Kids [2 0 R] >>
	//endobj
	//4 0 obj
	//<< /Type /Catalog /Pages 3 0 R >>
	//endobj
	//xref
	//0 5
	//0000000000 65535 f
	//0000000009 00000 n
	//0000000073 00000 n
	//0000000241 00000 n
	//0000000298 00000 n
	//trailer
	//<< /Root 4 0 R /Size 4 >>
	//starxref
	//347
	//%%EOF`)
}

// func TestPDFPath(t *testing.T) {
// 	buf := &bytes.Buffer{}
// 	pdf := newPDFWriter(buf).NewPage(210.0, 297.0)
// 	pdf.SetAlpha(0.5)
// 	pdf.SetFill(canvas.Paint{Color: canvas.Red})
// 	pdf.SetStroke(canvas.Paint{Color: canvas.Blue})
// 	pdf.SetLineWidth(5.0)
// 	pdf.SetLineCap(canvas.RoundCap)
// 	pdf.SetLineJoin(canvas.RoundJoin)
// 	pdf.SetDashes(2.0, []float64{1.0, 2.0, 3.0})
// 	test.String(t, pdf.String(), " 2.8346457 0 0 2.8346457 0 0 cm /A0 gs 1 0 0 rg /A1 gs 0 0 1 RG 5 w 1 J 1 j [1 2 3 1 2 3] 2 d")
// }
