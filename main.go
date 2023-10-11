package main

import (
	"fmt"
	"log"
	"math"

	"github.com/crazy3lf/colorconv"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	DRAW_PARTICLES      = false
	OPACITY             = 0.5
	WIDTH               = 256
	HEIGHT              = 256
	SCALE               = 4
	PARTICLE_COUNT      = 6000
	EMISSION_STRENGTH   = 15 / 144.0
	THREADS             = 8
	GRAVITY             = 0 / 144.0
	CENTER_GRAVITY      = 0.1 / 144.0
	MAX_CHARGE_RENDERED = 65
	NEGATIVE_RATE       = 0.5
	PALETTE_SIZE        = 10000
)

var (
	renderTime = 0.0
	simTime    = 0.0
	colorTable = [PALETTE_SIZE]FloatColor{}
)

type FloatColor struct {
	r, g, b float64
}

type Cell struct {
	charge float64
}

type Particle struct {
	x, y, xv, yv, charge float64
}

//lint:ignore U1000 bug
type FogCell struct {
	r, g, b, float64,
	hitRate float64
}

//lint:ignore U1000 might need later
func (p *FogCell) toBytes() (r, g, b uint8) {
	return uint8(p.r * 255 * p.hitRate), uint8(p.g * 255 * p.hitRate), uint8(p.b * 255 * p.hitRate)
}

func boundsCheck(x, y int) bool {
	return x >= 0 && x < WIDTH && y >= 0 && y < HEIGHT
}

func genColorTable() {
	for i := range colorTable {
		charge := float64(i) / PALETTE_SIZE
		hue := math.Mod(charge*360*50, 360)
		r, g, b, _ := colorconv.HSVToRGB(hue, 1, 1)
		colorTable[i] = FloatColor{float64(r) / 255, float64(g) / 255, float64(b) / 255}
	}
}

func getColor(charge float64) (r, g, b float64) {
	charge = min(1, math.Abs(charge))
	c := &colorTable[int(charge*PALETTE_SIZE)]
	return c.r, c.g, c.b
}

func main() {
	fmt.Println("Program started")
	genColorTable()
	scene := newScene()
	fmt.Println("Scene set up, running...")
	ebiten.SetWindowSize(WIDTH*SCALE, HEIGHT*SCALE)
	ebiten.SetWindowTitle("Simulation")
	if err := ebiten.RunGame(scene); err != nil {
		log.Fatal(err)
	}
}
