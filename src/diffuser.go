package main

type Diffuser struct {
	// The simulation is not in-place, so we need a swap buffer
	cellsIn  *[WIDTH][HEIGHT]Cell
	cellsOut *[WIDTH][HEIGHT]Cell
}

func newDiffuser() Diffuser {
	d := Diffuser{}
	d.cellsIn = &[WIDTH][HEIGHT]Cell{}
	d.cellsOut = &[WIDTH][HEIGHT]Cell{}
	return d
}

// Diffuses the charge of each cell to its neighbors.
// Also does exponential decay so the charge doesn't just keep
// going up or down forever.
func (d *Diffuser) diffusion(x, y int) {
	d.cellsOut[x][y].value = 0
	count := 0
	for sx := x - 1; sx <= x+1; sx++ {
		for sy := y - 1; sy <= y+1; sy++ {
			if sx >= 0 && sx < WIDTH && sy >= 0 && sy < HEIGHT {
				count++
				d.cellsOut[x][y].value += d.cellsIn[sx][sy].value
			}
		}
	}
	d.cellsOut[x][y].value /= float64(count)
	d.cellsOut[x][y].value *= 0.999
}

func (d *Diffuser) swapBuffers() {
	tmp := d.cellsIn
	d.cellsIn = d.cellsOut
	d.cellsOut = tmp
}
