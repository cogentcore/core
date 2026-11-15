// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/shape"
	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// phongTest is a Phong system rendering to a render texture.
type phongTest struct {
	gp *gpu.GPU
	rt *gpu.RenderTexture
	ph *Phong
}

// newPhongTest returns a new [phongTest] with a camera looking at the
// origin from z = 5.
func newPhongTest(t *testing.T) *phongTest {
	gpu.SkipIfNoGPU(t)
	pt := &phongTest{}
	gp, dev, err := gpu.NoDisplayGPU()
	require.NoError(t, err)
	pt.gp = gp
	pt.rt = gpu.NewRenderTexture(gp, dev, image.Pt(120, 90), 1, gpu.Depth32)
	pt.ph = NewPhong(gp, pt.rt)
	pt.ph.System.SetClearColor(color.Black)
	view := CameraViewMat(math32.Vec3(0, 0, 5), math32.Vec3(0, 0, 0), math32.Vec3(0, 1, 0))
	var proj math32.Matrix4
	proj.SetPerspective(45, pt.rt.Render().Format.Aspect(), 0.01, 100)
	pt.ph.SetCamera(view, &proj)
	return pt
}

func (pt *phongTest) release() {
	pt.ph.Release()
	pt.rt.Release()
	pt.gp.Release()
}

// render renders the given objects, each a mesh, object, and texture
// name (or ""), and returns the image.
func (pt *phongTest) render(t *testing.T, obs ...[3]string) *image.NRGBA {
	rp, err := pt.ph.RenderStart()
	require.NoError(t, err)
	for _, ob := range obs {
		require.NoError(t, pt.ph.UseObject(ob[1]))
		require.NoError(t, pt.ph.UseMesh(ob[0]))
		if ob[2] != "" {
			require.NoError(t, pt.ph.UseTexture(ob[2]))
		} else {
			pt.ph.UseNoTexture()
		}
		pt.ph.Render(rp)
	}
	img := pt.ph.RenderEndGrabImage(rp)
	require.NotNil(t, img)
	return img
}

// pixel returns the color at the given position.
func pixel(img image.Image, x, y int) color.RGBA {
	return color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
}

// dominant asserts that the given channel (0 = R, 1 = G, 2 = B) of the
// color is clearly larger than the others.
func dominant(t *testing.T, c color.RGBA, ch int, msg string) {
	t.Helper()
	v := []uint8{c.R, c.G, c.B}
	for i, o := range v {
		if i != ch {
			assert.Greater(t, int(v[ch]), int(o)+60, "%s: %v", msg, c)
		}
	}
}

// brightness returns the sum of the color channels.
func brightness(c color.RGBA) int {
	return int(c.R) + int(c.G) + int(c.B)
}

// TestPhongColors tests rendering objects with colors under ambient
// light, with the background shown where there is no object.
func TestPhongColors(t *testing.T) {
	pt := newPhongTest(t)
	defer pt.release()
	ph := pt.ph
	ph.AddAmbient(math32.Vec3(1, 1, 1))
	ph.SetMesh("box", shape.NewBox(1, 1, 1))
	var left, right math32.Matrix4
	left.SetTranslation(-1, 0, 0)
	right.SetTranslation(1, 0, 0)
	ph.SetObject("red", NewObject(&left, NewColors(color.RGBA{255, 0, 0, 255}, color.Black, 30, 0, 1)))
	ph.SetObject("blue", NewObject(&right, NewColors(color.RGBA{0, 0, 255, 255}, color.Black, 30, 0, 1)))

	img := pt.render(t, [3]string{"box", "red", ""}, [3]string{"box", "blue", ""})
	dominant(t, pixel(img, 36, 45), 0, "red box on the left")
	dominant(t, pixel(img, 84, 45), 2, "blue box on the right")
	assert.Equal(t, color.RGBA{0, 0, 0, 255}, pixel(img, 60, 5), "background")

	assert.Error(t, ph.UseMesh("none"))
	assert.Error(t, ph.UseObject("none"))
	assert.Error(t, ph.UseTexture("none"))
}

// TestPhongLights tests that a directional light makes a surface facing
// it brighter than ambient light alone, and that point and spot lights
// add light.
func TestPhongLights(t *testing.T) {
	pt := newPhongTest(t)
	defer pt.release()
	ph := pt.ph
	ph.SetMesh("plane", shape.NewPlane(math32.Z, 2, 2))
	var mtx math32.Matrix4
	mtx.SetIdentity()
	ph.SetObject("plane", NewObject(&mtx, NewColors(color.RGBA{200, 200, 200, 255}, color.Black, 30, 0, 1)))
	center := func() color.RGBA {
		return pixel(pt.render(t, [3]string{"plane", "plane", ""}), 60, 45)
	}

	ph.AddAmbient(math32.Vec3(0.2, 0.2, 0.2))
	ambient := brightness(center())
	assert.Greater(t, ambient, 0, "lit by ambient light")

	ph.AddDirectional(math32.Vec3(1, 1, 1), math32.Vec3(0, 0, 1))
	assert.Equal(t, int32(1), ph.NLights.Directional)
	direct := brightness(center())
	assert.Greater(t, direct, ambient+50, "a light in front makes it brighter")

	ph.SetDirectional(0, math32.Vec3(1, 1, 1), math32.Vec3(0, 0, -1)) // behind
	assert.Less(t, brightness(center()), direct, "a light behind does not")

	ph.ResetLights()
	ph.AddAmbient(math32.Vec3(0.2, 0.2, 0.2))
	ph.AddPoint(math32.Vec3(1, 1, 1), math32.Vec3(0, 0, 2), 0.01, 0.001)
	assert.Greater(t, brightness(center()), ambient+50, "point light")

	ph.ResetLights()
	ph.AddAmbient(math32.Vec3(0.2, 0.2, 0.2))
	ph.AddSpot(math32.Vec3(1, 1, 1), math32.Vec3(0, 0, 2), math32.Vec3(0, 0, -1), 1, 45, 0.01, 0.001)
	assert.Greater(t, brightness(center()), ambient+50, "spot light")
	ph.SetSpot(0, math32.Vec3(0, 0, 0), math32.Vec3(0, 0, 2), math32.Vec3(0, 0, -1), 1, 45, 0.01, 0.001)
	ph.SetPoint(0, math32.Vec3(0, 0, 0), math32.Vec3(0, 0, 2), 0.01, 0.001)
	ph.SetAmbient(0, math32.Vec3(0, 0, 0))
	assert.Equal(t, color.RGBA{0, 0, 0, 255}, center(), "no light")
}

// TestPhongEmissive tests that an emissive color shows without light.
func TestPhongEmissive(t *testing.T) {
	pt := newPhongTest(t)
	defer pt.release()
	ph := pt.ph
	ph.SetMesh("sphere", shape.NewSphere(1, 32))
	var mtx math32.Matrix4
	mtx.SetIdentity()
	ph.SetObject("glow", NewObject(&mtx, NewColors(color.Black, color.RGBA{0, 255, 0, 255}, 30, 0, 1)))
	ph.ConfigLights()
	dominant(t, pixel(pt.render(t, [3]string{"sphere", "glow", ""}), 60, 45), 1, "emissive green")
}

// TestPhongTexture tests rendering an object with a texture, which
// overrides its color, and replacing the texture.
func TestPhongTexture(t *testing.T) {
	pt := newPhongTest(t)
	defer pt.release()
	ph := pt.ph
	ph.AddAmbient(math32.Vec3(1, 1, 1))
	ph.SetMesh("plane", shape.NewPlane(math32.Z, 2, 2))
	var mtx math32.Matrix4
	mtx.SetIdentity()
	ob := NewObject(&mtx, NewColors(color.White, color.Black, 30, 0, 1))
	ob.UseFullTexture()
	ph.SetObject("plane", ob)

	solid := func(c color.RGBA) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, 8, 8))
		draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
		return img
	}
	ph.SetTexture("green", NewTexture(solid(color.RGBA{0, 255, 0, 255})))
	ph.SetTexture("blue", NewTexture(solid(color.RGBA{0, 0, 255, 255})))
	dominant(t, pixel(pt.render(t, [3]string{"plane", "plane", "green"}), 60, 45), 1, "green texture")
	dominant(t, pixel(pt.render(t, [3]string{"plane", "plane", "blue"}), 60, 45), 2, "blue texture")

	ph.SetTexture("green", NewTexture(solid(color.RGBA{255, 0, 0, 255})))
	dominant(t, pixel(pt.render(t, [3]string{"plane", "plane", "green"}), 60, 45), 0, "replaced texture")
	assert.Greater(t, brightness(pixel(pt.render(t, [3]string{"plane", "plane", ""}), 60, 45)), 3*240, "without a texture, the color of the object (white)")

	ob2 := NewObject(&mtx, NewColors(color.White, color.Black, 30, 0, 1))
	ob2.SetTilingRepeat(math32.Vec2(2, 2)).SetTilingOffset(math32.Vec2(0.5, 0))
	assert.Equal(t, math32.Vec2(2, 2), ob2.Tiling.Repeat)
	ph.ResetAll()
}

// TestPhongMeshes tests rendering each of the standard meshes, which
// all cover the center of the view.
func TestPhongMeshes(t *testing.T) {
	pt := newPhongTest(t)
	defer pt.release()
	ph := pt.ph
	ph.AddAmbient(math32.Vec3(1, 1, 1))
	meshes := map[string]shape.Mesh{
		"box":      shape.NewBox(1, 1, 1),
		"sphere":   shape.NewSphere(0.6, 24),
		"cylinder": shape.NewCylinder(1, 0.5, 24, 1, true, true),
		"cone":     shape.NewCone(1, 0.5, 24, 1, true),
		"capsule":  shape.NewCapsule(1, 0.4, 24, 8),
		"torus":    shape.NewTorus(1, 0.3, 24),
		"lines":    shape.NewLines([]math32.Vector3{{X: -1, Y: 0, Z: 0}, {X: 1, Y: 0, Z: 0}}, math32.Vec2(0.3, 0.3), false),
	}
	var mtx math32.Matrix4
	mtx.SetIdentity()
	ph.SetObject("ob", NewObject(&mtx, NewColors(color.RGBA{255, 0, 0, 255}, color.Black, 30, 0, 1)))
	for name, m := range meshes {
		ph.SetMesh(name, m)
	}
	for name := range meshes {
		img := pt.render(t, [3]string{name, "ob", ""})
		x, y := 60, 45
		if name == "torus" { // its hole is in the center
			x = 60 + 25
		}
		dominant(t, pixel(img, x, y), 0, name)
		assert.Equal(t, color.RGBA{0, 0, 0, 255}, pixel(img, 2, 2), name+" background")
	}
	ph.ResetMeshes()
	assert.Error(t, ph.UseMesh("box"))
}
