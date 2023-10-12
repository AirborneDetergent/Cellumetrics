package main

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/lucasb-eyer/go-colorful"
)

const (
	DRAW_PARTICLES = false
	// Particles are alpha-blended onto the final image when rendered
	OPACITY = 0.5
	// This is for the diffusion simulation, but also affects window size
	WIDTH  = 256
	HEIGHT = 256
	// Makes the window bigger because the simulation is kinda small
	SCALE          = 4
	PARTICLE_COUNT = 10000
	// What portion of particles should be negatively charged
	NEGATIVE_RATE = 0.5
	// What portion of particles should be exceptionally bright
	BRIGHT_RATE = 0.0
	// How much brighter to make bright particles
	STR_BOOST = 10
	// How much charge each particle should produce
	CHARGE_STRENGTH = 8 / 144.0
	// How much colored fog each particle should produce.
	// Only seems to make a small difference.
	EMISSION_STRENGTH = 10 / 144.0
	THREADS           = 8
	// Allows both gravity that pulls particles towards the center and
	// gravity that pulls particles down. They can't exit the window.
	GRAVITY        = 0 / 144.0
	CENTER_GRAVITY = 0.1 / 144.0
	// Only affects rendering. Will affect the overall density of the fog
	// and colors chosen based on density
	CHARGE_DIVISOR = 65
	// Uses a precomputed HSV hue wheel because it can be slow
	PALETTE_SIZE = 1000
	// Brightness
	INIT_RAY_LUMA = 70
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
	value float64
}

// charge is only ever -1 or 1 and affects what the
// particles emit as well as how they move
type Particle struct {
	x, y, xv, yv, charge, r, g, b float64
	isBright                      bool
}

// hitRate is meant to describe, on average, how much
// light would end up hitting a fog particle when
// passing through that cell. [0, 1)
type FogCell struct {
	r, g, b, hitRate float64
}

func main() {
	genColorTable()
	scene := newScene()
	ebiten.SetWindowSize(WIDTH*SCALE, HEIGHT*SCALE)
	ebiten.SetWindowTitle("Simulation")
	if err := ebiten.RunGame(scene); err != nil {
		log.Fatal(err)
	}
}

func boundsCheck(x, y int) bool {
	return x >= 0 && x < WIDTH && y >= 0 && y < HEIGHT
}

func genColorTable() {
	for i := range colorTable {
		charge := float64(i) / PALETTE_SIZE
		hue := math.Mod(charge*360, 360)
		col := colorful.Hsv(hue, 1, 1)
		colorTable[i] = FloatColor{col.R, col.G, col.B}
	}
}

// Negative values will go the other direction,
// but this function will clamp if given a value
// not within [-1, 1]
func getColor(charge float64) (r, g, b float64) {
	place := min(1, math.Abs(charge))
	if charge < 0 {
		place = 1 - place
	}
	c := &colorTable[int(place*(PALETTE_SIZE-1))]
	return c.r, c.g, c.b
}

// Muh CPU can't handle 2.2. Rendering does more work now,
// but when I implemented this, it made the entire step >3x slower.
func gammaCorrect(n float64) float64 {
	return n * n
}

// This is very slow, but I'm not sure how to speed it up
func saturate(r, g, b, amt float64) (float64, float64, float64) {
	col := colorful.Color{R: r, G: g, B: b}
	h, s, v := col.Hsv()
	if s != 0 {
		s = min(1, s*amt)
	}
	v = (v-0.5)/amt + 0.5
	col = colorful.Hsv(h, s, v)
	return col.R, col.G, col.B
}
