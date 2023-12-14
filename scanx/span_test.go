package scanx_test

import (
	"bufio"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/xgbutil/xgraphics"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/scanFT"
	"github.com/srwiley/scanx"

	"github.com/srwiley/rasterx"
)

func SaveToPngFile(filePath string, m image.Image) error {
	// Create the file
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	// Create Writer from file
	b := bufio.NewWriter(f)
	// Write the image into the buffer
	err = png.Encode(b, m)
	if err != nil {
		return err
	}
	err = b.Flush()
	if err != nil {
		return err
	}
	return nil
}

func FilePathWalkDir(root string) (files []string, err error) {
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) (e error) {
		if !info.IsDir() && strings.HasSuffix(path, ".svg") {
			files = append(files, path)
		}
		return
	})
	return
}

func Clear(img *image.RGBA) {
	for b := 0; b < len(img.Pix); b++ {
		img.Pix[b] = 0
	}
}
func ClearX(img *xgraphics.Image) {
	for b := 0; b < len(img.Pix); b++ {
		img.Pix[b] = 0
	}
}

func CompareSpanners(t *testing.T, file string, img1, img2 *image.RGBA, width, height int, op draw.Op, vsFT, testImg bool, dLimit int) {
	Clear(img1)
	Clear(img2)

	if strings.HasSuffix(file, "gradient.svg") && testImg == false {
		t.Log("skipping gradient example for compress spanner")
		return
	}

	icon, errSvg := oksvg.ReadIcon(file, oksvg.WarnErrorMode)

	if errSvg != nil {
		fmt.Println("cannot read icon")
		log.Fatal("cannot read icon", errSvg)
		t.FailNow()
	}

	icon.SetTarget(float64(0), float64(0), float64(width), float64(height))
	if testImg {
		spanner := scanx.NewImgSpanner(img1)
		spanner.Op = op
		scannerX := scanx.NewScanner(spanner, width, height)
		rasterScanX := rasterx.NewDasher(width, height, scannerX)
		icon.Draw(rasterScanX, 1.0)
	} else {
		spannerC := &scanx.LinkListSpanner{}
		spannerC.Op = op
		spannerC.SetBounds(image.Rect(0, 0, width, height))
		scannerC := scanx.NewScanner(spannerC, width, height)
		rasterScanC := rasterx.NewDasher(width, height, scannerC)
		icon.Draw(rasterScanC, 1.0)
		spannerC.DrawToImage(img1)
	}
	if vsFT {
		painter := scanFT.NewRGBAPainter(img2)
		scanner := scanFT.NewScannerFT(width, height, painter)
		rasterFT := rasterx.NewDasher(width, height, scanner)
		icon.Draw(rasterFT, 1.0)
	} else {
		spanner := scanx.NewImgSpanner(img2)
		spanner.Op = op
		scannerX := scanx.NewScanner(spanner, width, height)
		rasterScanX := rasterx.NewDasher(width, height, scannerX)
		icon.Draw(rasterScanX, 1.0)
	}

	var pix1 = img1.Pix
	var pix2 = img2.Pix
	var stride = img2.Stride

	if len(pix1) == 0 {
		t.Error("images are zero sized ")
		t.FailNow()
	}
	i0 := 0
	maxd := 0
	for y := 0; y < img2.Bounds().Max.Y; y++ {
		i0 = y * stride
		for x := 0; x < img2.Bounds().Max.X*4; x += 4 {
			for k := 0; k < 4; k++ {
				d := int(pix1[i0+x+k]) - int(pix2[i0+x+k])
				if d < -maxd {
					maxd = -d
				} else if d > maxd {
					maxd = d
				}
				if d < -dLimit || d > dLimit {
					SaveToPngFile("testdata/img1.png", img1)
					SaveToPngFile("testdata/img2.png", img2)
					t.Error("image comparison failed for file ", file)
					t.Error("images do not match at index ", d, k, y, x/4,
						"c1", pix1[i0+x], pix1[i0+x+1], pix1[i0+x+2], pix1[i0+x+3],
						"c2", pix2[i0+x], pix2[i0+x+1], pix2[i0+x+2], pix2[i0+x+3])
					t.FailNow()
				}
			}
		}
	}
	//t.Log("maxd ", maxd)
}

func CompareSpannersX(t *testing.T, file string, img1, img2 *xgraphics.Image, width, height int, op draw.Op, testImg bool, dLimit int) {
	ClearX(img1)
	ClearX(img2)

	if strings.HasSuffix(file, "gradient.svg") && testImg == false {
		t.Log("skipping gradient example for compress spanner")
		return
	}

	icon, errSvg := oksvg.ReadIcon(file, oksvg.WarnErrorMode)

	if errSvg != nil {
		fmt.Println("cannot read icon")
		log.Fatal("cannot read icon", errSvg)
		t.FailNow()
	}

	icon.SetTarget(float64(0), float64(0), float64(width), float64(height))

	if testImg {
		spanner := scanx.NewImgSpanner(img1)
		spanner.Op = op
		scannerX := scanx.NewScanner(spanner, width, height)
		rasterScanX := rasterx.NewDasher(width, height, scannerX)
		icon.Draw(rasterScanX, 1.0)
	} else {
		spannerC := &scanx.LinkListSpanner{}
		spannerC.Op = op
		spannerC.SetBounds(image.Rect(0, 0, width, height))
		scannerC := scanx.NewScanner(spannerC, width, height)
		rasterScanC := rasterx.NewDasher(width, height, scannerC)
		icon.Draw(rasterScanC, 1.0)
		spannerC.DrawToImage(img1)
	}

	spanner := scanx.NewImgSpanner(img2)
	spanner.Op = op
	scannerX := scanx.NewScanner(spanner, width, height)
	rasterScanX := rasterx.NewDasher(width, height, scannerX)
	icon.Draw(rasterScanX, 1.0)

	var pix1 = img1.Pix
	var pix2 = img2.Pix
	var stride = img2.Stride

	if len(pix1) == 0 {
		t.Error("images are zero sized ")
		t.FailNow()
	}
	i0 := 0
	maxd := 0
	for y := 0; y < img2.Bounds().Max.Y; y++ {
		i0 = y * stride
		for x := 0; x < img2.Bounds().Max.X*4; x += 4 {
			for k := 0; k < 4; k++ {
				d := int(pix1[i0+x+k]) - int(pix2[i0+x+k])
				if d < -maxd {
					maxd = -d
				} else if d > maxd {
					maxd = d
				}
				if d < -dLimit || d > dLimit {
					SaveToPngFile("testdata/img1.png", img1)
					SaveToPngFile("testdata/img2.png", img2)
					t.Error("image comparison failed for file ", file)
					t.Error("images do not match at index ", d, k, y, x/4,
						"c1", pix1[i0+x], pix1[i0+x+1], pix1[i0+x+2], pix1[i0+x+3],
						"c2", pix2[i0+x], pix2[i0+x+1], pix2[i0+x+2], pix2[i0+x+3])
					t.FailNow()
				}
			}
		}
	}
}

func TestSpannersImg(t *testing.T) {
	width := 400
	height := 350

	img1 := image.NewRGBA(image.Rect(0, 0, width, height))
	img2 := image.NewRGBA(image.Rect(0, 0, width, height))

	svgs, err := FilePathWalkDir("testdata/svg")
	if err != nil {
		fmt.Println("cannot walk file path testdata/svg")
		t.FailNow()
	}
	for _, f := range svgs {
		CompareSpanners(t, f, img1, img2, width, height, draw.Src, false, false, 0)
	}
	for _, f := range svgs {
		CompareSpanners(t, f, img1, img2, width, height, draw.Over, false, false, 0)
	}
	for _, f := range svgs {
		CompareSpanners(t, f, img1, img2, width, height, draw.Over, true, false, 4)
	}
	for _, f := range svgs {
		CompareSpanners(t, f, img1, img2, width, height, draw.Over, true, true, 4)
	}
}

func TestSpannersX(t *testing.T) {
	width := 400
	height := 350

	ximgx := xgraphics.New(nil, image.Rect(0, 0, width, height))
	ximgc := xgraphics.New(nil, image.Rect(0, 0, width, height))
	svgs, err := FilePathWalkDir("testdata/svg")
	if err != nil {
		fmt.Println("cannot walk file path testdata/svg")
		t.FailNow()
	}
	for _, f := range svgs {
		CompareSpannersX(t, f, ximgx, ximgc, width, height, draw.Over, true, 0)
		CompareSpannersX(t, f, ximgx, ximgc, width, height, draw.Over, false, 0)
		CompareSpannersX(t, f, ximgx, ximgc, width, height, draw.Src, false, 0)
		CompareSpannersX(t, f, ximgx, ximgc, width, height, draw.Src, true, 0)
	}
}

// func TestCompose(t *testing.T) {
// 	spannerC := &scanx.LinkListSpanner{}
// 	spannerC.SetBounds(image.Rect(0, 0, 10, 10))
// 	spannerC.TestSpanAdd()

// }

// func TestCompose(t *testing.T) {
// 	sp := &scanx.LinkListSpanner{}
// 	fmt.Println("cells, index", len(sp.spans))
// 	sp.SetBounds(image.Rect(0, 0, 100, 2))
// 	fmt.Println("cells, index", len(sp.spans))
// 	sp.SetBounds(image.Rect(0, 0, 100, 1))
// 	fmt.Println("cells, index", len(sp.spans))

// 	drawList := func(y int) {
// 		fmt.Print("list at ", y, ":")
// 		cntr := 0
// 		p := sp.spans[y].next
// 		for p != 0 {
// 			spCell := sp.spans[p]
// 			fmt.Print(" sp", spCell)
// 			p = spCell.next
// 			cntr++
// 		}
// 		fmt.Println(" length", cntr)
// 	}

// 	drawList(0)

// 	sp.SetBgColor(color.Black)
// 	sp.SpanOver(0, 20, 40, m)

// 	drawList(0)

// 	sp.SpanOver(0, 5, 10, m)
// 	drawList(0)

// 	sp.SpanOver(0, 80, 90, m)
// 	drawList(0)

// 	sp.SpanOver(0, 60, 70, m)
// 	drawList(0)
// 	sp.SpanOver(0, 70, 75, m)
// 	drawList(0)

// 	sp.Clear()
// 	drawList(0)

// 	sp.SpanOver(0, 20, 40, m)
// 	drawList(0)

// 	sp.SpanOver(0, 30, 50, m)
// 	drawList(0)

// 	sp.SpanOver(0, 10, 25, m)
// 	drawList(0)
// 	sp.SpanOver(0, 33, 37, m)
// 	drawList(0)

// 	sp.Clear()
// 	drawList(0)

// 	sp.SpanOver(0, 20, 40, m)
// 	drawList(0)

// 	sp.SpanOver(0, 20, 40, m)
// 	drawList(0)

// 	sp.SpanOver(0, 10, 40, m)
// 	drawList(0)

// 	sp.SpanOver(0, 20, 50, m)
// 	drawList(0)

// 	sp.Clear()
// 	drawList(0)

// 	sp.SpanOver(0, 20, 30, m)
// 	drawList(0)

// 	sp.SpanOver(0, 40, 50, m)
// 	drawList(0)

// 	sp.SpanOver(0, 10, 60, m)
// 	drawList(0)
// }

// func (x *LinkListSpanner) CheckList(y int, m string) {
// 	cntr := 0
// 	p := x.spans[y].next
// 	for p != 0 {
// 		spCell := x.spans[p]
// 		//fmt.Print(" sp", spCell)
// 		p = spCell.next
// 		if p != 0 {
// 			snCell := x.spans[p]
// 			if spCell.x1 > snCell.x0 {
// 				fmt.Println("bad list at ", y, ":", m)
// 				x.DrawList(y)
// 				os.Exit(1)
// 			}
// 		}
// 		cntr++
// 	}
// }

// //DrawList draws the linked list y
// func (x *LinkListSpanner) DrawList(y int) {
// 	fmt.Print("list at ", y, ":")
// 	cntr := 0
// 	p := x.spans[y].next
// 	for p != 0 {
// 		spCell := x.spans[p]
// 		fmt.Print(" ", p, ":sp", spCell)
// 		p = spCell.next
// 		cntr++
// 	}
// 	fmt.Println(" length", cntr)
// }
