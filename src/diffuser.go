package main

import "math"

type Diffuser struct {
	// The simulation is not in-place, so we need a swap buffer
	cellsIn   *[WIDTH][HEIGHT]Cell
	cellsOut  *[WIDTH][HEIGHT]Cell
	kernel    [9]float64
	decayCoef float64
}

// Retainment describes how much stuff stays in its cell
// each iteration instead of diffusing to each neighbor.
// Decay describes how quickly exponential decay occurs.
func newDiffuser(retainment, decay float64) Diffuser {
	d := Diffuser{}
	d.cellsIn = &[WIDTH][HEIGHT]Cell{}
	d.cellsOut = &[WIDTH][HEIGHT]Cell{}
	d.kernel = [9]float64{
		1, 1, 1,
		1, retainment, 1,
		1, 1, 1,
	}
	d.decayCoef = math.Pow(0.999, decay)
	return d
}

// Diffuses the charge of each cell to its neighbors.
// Also does exponential decay so the charge doesn't just keep
// going up or down forever.
func (d *Diffuser) diffusion(x, y int) {
	d.cellsOut[x][y].value = 0
	sum := 0.0
	index := 0
	for sx := x - 1; sx <= x+1; sx++ {
		for sy := y - 1; sy <= y+1; sy++ {
			if boundsCheck(sx, sy) {
				kv := d.kernel[index]
				d.cellsOut[x][y].value += d.cellsIn[sx][sy].value * kv
				sum += kv
				index++
			}
		}
	}
	d.cellsOut[x][y].value /= sum
	d.cellsOut[x][y].value *= d.decayCoef
}

func (d *Diffuser) swapBuffers() {
	tmp := d.cellsIn
	d.cellsIn = d.cellsOut
	d.cellsOut = tmp
}
