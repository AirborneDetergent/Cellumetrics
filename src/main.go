package main

import (
	"log"
	"math"

	"github.com/crazy3lf/colorconv"
	"github.com/hajimehoshi/ebiten/v2"
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
	PARTICLE_COUNT = 6000
	// How much charge (and also fog, but this affects the simulation)
	// each particle should produce
	EMISSION_STRENGTH = 15 / 144.0
	THREADS           = 8
	// Allows both gravity that pulls particles towards the center and
	// gravity that pulls particles down. They can't exit the window.
	GRAVITY        = 0 / 144.0
	CENTER_GRAVITY = 0.1 / 144.0
	// Only affects rendering. Will affect the overall density of the fog
	// and colors chosen based on density
	CHARGE_DIVISOR = 50
	// What portion of particles should be negatively charged
	NEGATIVE_RATE = 0.5
	// Uses a precomputed HSV hue wheel because it can be slow
	PALETTE_SIZE = 1000
	// Brightness
	INIT_RAY_LUMA = 75
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
}

// hitRate is meant to describe, on average, how much
// light would end up hitting a fog particle when
// passing through that cell. [0, 1)
type FogCell struct {
	r, g, b, hitRate float64
}

func boundsCheck(x, y int) bool {
	return x >= 0 && x < WIDTH && y >= 0 && y < HEIGHT
}

func genColorTable() {
	for i := range colorTable {
		charge := float64(i) / PALETTE_SIZE
		hue := math.Mod(charge*360, 360)
		r, g, b, _ := colorconv.HSVToRGB(hue, 1, 1)
		colorTable[i] = FloatColor{float64(r) / 255, float64(g) / 255, float64(b) / 255}
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

func main() {
	genColorTable()
	scene := newScene()
	ebiten.SetWindowSize(WIDTH*SCALE, HEIGHT*SCALE)
	ebiten.SetWindowTitle("Simulation")
	if err := ebiten.RunGame(scene); err != nil {
		log.Fatal(err)
	}
}

func signedSmoothClamp(x float64) float64 {
	if x == 0 {
		return 0
	}
	return math.Copysign(1-1.0/(math.Abs(x)+1), x)
}
