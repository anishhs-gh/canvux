package imgembed

import (
	"image"
	"image/color"
	"testing"
)

func TestConvertMergesRuns(t *testing.T) {
	// 8x4 image: top half solid red, bottom half solid blue.
	img := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			c := color.RGBA{R: 200, A: 255}
			if y >= 2 {
				c = color.RGBA{B: 200, A: 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	objs := Convert(img, 8, 0)
	// Perfect run-merging: one rect per row = 4 objects.
	if len(objs) != 4 {
		t.Fatalf("Convert produced %d objects, want 4 merged row-runs", len(objs))
	}
	for _, o := range objs {
		if !o.Filled || o.Bounds().W() != 8 {
			t.Errorf("run not merged across the row: %+v", o)
		}
	}
}

func TestConvertSkipsTransparent(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4)) // fully transparent
	if objs := Convert(img, 4, 0); len(objs) != 0 {
		t.Errorf("transparent image produced %d objects", len(objs))
	}
}

func TestConvertCapsSize(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4000, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 4000; x++ {
			img.SetRGBA(x, y, color.RGBA{G: 128, A: 255})
		}
	}
	objs := Convert(img, 48, 0)
	for _, o := range objs {
		if o.Bounds().Max.X > 48 {
			t.Fatalf("object exceeds column cap: %+v", o.Bounds())
		}
	}
	if len(objs) == 0 {
		t.Fatal("wide image produced nothing")
	}
}
