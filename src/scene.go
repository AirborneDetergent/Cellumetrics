package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

type Scene struct {
	// Raw data sent to the GPU
	frameBuffer    []byte
	fogBuffer      [WIDTH][HEIGHT]FogCell
	colorDiffusers [3]Diffuser
	chargeDiffuser Diffuser
	particles      [PARTICLE_COUNT]Particle
	// We need a swap buffer for light ray blending
	lightBufferIn  *[WIDTH]FloatColor
	lightBufferOut *[WIDTH]FloatColor
}

func newScene() *Scene {
	s := &Scene{}
	s.frameBuffer = make([]byte, WIDTH*HEIGHT*4)
	s.chargeDiffuser = newDiffuser(1, 1)
	for i := range s.colorDiffusers {
		s.colorDiffusers[i] = newDiffuser(1, 10)
	}
	s.lightBufferIn = &[WIDTH]FloatColor{}
	s.lightBufferOut = &[WIDTH]FloatColor{}
	// Particles are initialized with random positions
	// and randomly signed charges
	for i := range s.particles {
		p := &s.particles[i]
		p.x = rand.Float64() * WIDTH
		p.y = rand.Float64() * HEIGHT
		if rand.Float64() < NEGATIVE_RATE {
			p.charge = -1
			p.r, p.g, p.b = getColor(rand.Float64()*0.15 - 0.075 + 0.1)
		} else {
			p.charge = 1
			p.r, p.g, p.b = getColor(rand.Float64()*0.15 - 0.075 + 0.6)
		}
		p.isBright = rand.Float64() < BRIGHT_RATE

	}
	return s
}

// Prioritizes displaying every frame over
// running at a consistent speed, so this is unused
func (s *Scene) Update() error {
	return nil
}

func (s *Scene) Draw(screen *ebiten.Image) {
	debugInfo := ""
	timer := makeTimer()
	s.simStep()
	// Exponential moving average so the number isn't super spazzy and useless
	simTime = simTime*0.9 + timer.tick()*0.1
	s.render(screen)
	renderTime = renderTime*0.9 + timer.tick()*0.1
	debugInfo += fmt.Sprintf("FPS: %0.4g\n", ebiten.ActualFPS())
	debugInfo += fmt.Sprintf("Simulation time: %0.3f\n", simTime)
	debugInfo += fmt.Sprintf("Render time: %0.3f\n", renderTime)
	//ebitenutil.DebugPrint(screen, debugInfo)
}

func (s *Scene) readChargeBounded(x, y int) float64 {
	if boundsCheck(x, y) {
		return s.chargeDiffuser.cellsIn[x][y].value
	}
	return 0.0
}

// Estimates the derivative of the charge at (x, y)
func (s *Scene) approxDeriv(x, y int) (dx, dy float64) {
	dx = s.readChargeBounded(x+1, y) - s.readChargeBounded(x-1, y)
	dy = s.readChargeBounded(x, y+1) - s.readChargeBounded(x, y-1)
	return dx, dy
}

// Calls a function with the x and y positions of every cell
// in parallel
func (s *Scene) perCellThreaded(funs ...func(int, int)) {
	wg := sync.WaitGroup{}
	wg.Add(THREADS)
	for i := 0; i < THREADS; i++ {
		go s.perCellThreadedHelper(i, &wg, funs)
	}
	wg.Wait()
}

func (s *Scene) perCellThreadedHelper(offset int, wg *sync.WaitGroup, funs []func(int, int)) {
	for x := offset; x < WIDTH; x += THREADS {
		for y := 0; y < HEIGHT; y++ {
			for _, fun := range funs {
				fun(x, y)
			}
		}
	}
	wg.Done()
}

func (s *Scene) simStep() {
	// Add charge to random locations around the mouse when clicking
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		for i := 0; i < 100; i++ {
			mouseX, mouseY := ebiten.CursorPosition()
			mouseX += rand.Intn(32) - 16
			mouseY += rand.Intn(32) - 16
			if !boundsCheck(mouseX, mouseY) {
				continue
			}
			s.chargeDiffuser.cellsIn[mouseX][mouseY].value += 0.1
		}

	}

	s.perCellThreaded(s.chargeDiffuser.diffusion,
		s.colorDiffusers[0].diffusion,
		s.colorDiffusers[1].diffusion,
		s.colorDiffusers[2].diffusion)

	// Simulate particle movement. Particles never interact directly
	// and can only affect each other through the cellular simulation,
	// so a swap buffer is not needed.
	for i := range s.particles {
		p := &s.particles[i]
		// Negatively charged particles will accelerate with the derivative
		// of the charge field at their current position, while positively
		// charged particles will accelerate in the opposite direction
		dx, dy := s.approxDeriv(int(p.x), int(p.y))
		p.xv -= dx / 144 * p.charge
		p.yv -= dy / 144 * p.charge
		// Gravity is applied as constant forces both downwards and
		// towards the center of the screen
		p.yv += GRAVITY
		if CENTER_GRAVITY > 0 {
			difX := p.x - WIDTH/2
			difY := p.y - HEIGHT/2
			length := math.Sqrt(difX*difX + difY*difY)
			modifier := min(1, length/32)
			modifier *= modifier
			p.xv -= difX / length * CENTER_GRAVITY * modifier
			p.yv -= difY / length * CENTER_GRAVITY * modifier
		}
		// This system doesn't exactly preserve energy, so this
		// exponential decay is necessary to prevent things from going crazy
		p.xv *= 0.995
		p.yv *= 0.995
		p.x += p.xv
		p.y += p.yv
		// Keep particles inside the window
		if p.x < 1 {
			p.x = 1
			p.xv = math.Max(0, p.xv)
		}
		if p.x >= WIDTH-1 {
			p.x = WIDTH - 2
			p.xv = math.Min(0, p.xv)
		}
		if p.y < 1 {
			p.y = 1
			p.yv = math.Max(0, p.yv)
		}
		if p.y >= HEIGHT-1 {
			p.y = HEIGHT - 2
			p.yv = math.Min(0, p.yv)
		}
	}
	// Positive and negative particles increase and decrease the charge of
	// the cell they're on respectively. They also all add to the color diffusers' cells.
	for i := range s.particles {
		p := &s.particles[i]
		ix, iy := int(p.x), int(p.y)
		if boundsCheck(ix, iy) {
			coef := 1.0
			if p.isBright {
				coef = STR_BOOST
			}
			s.chargeDiffuser.cellsOut[ix][iy].value += CHARGE_STRENGTH * p.charge * coef
			s.colorDiffusers[0].cellsOut[ix][iy].value += EMISSION_STRENGTH * p.r * coef
			s.colorDiffusers[1].cellsOut[ix][iy].value += EMISSION_STRENGTH * p.g * coef
			s.colorDiffusers[2].cellsOut[ix][iy].value += EMISSION_STRENGTH * p.b * coef
		}
	}
	s.chargeDiffuser.swapBuffers()
	s.colorDiffusers[0].swapBuffers()
	s.colorDiffusers[1].swapBuffers()
	s.colorDiffusers[2].swapBuffers()
}

func (s *Scene) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return WIDTH, HEIGHT
}

// Turn the cell charge values into colored fog
func (s *Scene) calcFog(x, y int) {
	cell := &s.chargeDiffuser.cellsIn[x][y]
	redCell := &s.colorDiffusers[0].cellsIn[x][y]
	greenCell := &s.colorDiffusers[1].cellsIn[x][y]
	blueCell := &s.colorDiffusers[2].cellsIn[x][y]
	charge := cell.value / CHARGE_DIVISOR
	fog := FogCell{}
	fog.r, fog.g, fog.b = redCell.value, greenCell.value, blueCell.value
	fog.r, fog.g, fog.b = saturate(fog.r, fog.g, fog.b, 2.)
	mag := math.Sqrt(fog.r*fog.r + fog.g*fog.g + fog.b*fog.b)
	if mag > 0 {
		fog.r /= mag
		fog.g /= mag
		fog.b /= mag

	}
	fog.hitRate = 1 - math.Pow(0.5, math.Abs(charge))
	s.fogBuffer[x][y] = fog
}

// Renders the fog computed in calcFog() to the screen with volumetrics.
// One ray of light is cast down from each column of pixels on the screen.
// This can't be multithreaded because every time the rays move a step down,
// their light values are also blended with their neighbors' light values.
func (s *Scene) drawFog() {
	// Initialize all the rays
	for i := range s.lightBufferIn {
		s.lightBufferIn[i] = FloatColor{INIT_RAY_LUMA, INIT_RAY_LUMA, INIT_RAY_LUMA}
	}
	for y := 0; y < HEIGHT; y++ {
		// Figure out how the light should mix with the current row's
		// fog cells and draw the colors on the screen while also
		// letting the fog absorb some of the light.
		for x := 0; x < WIDTH; x++ {
			light := &s.lightBufferIn[x]
			index := (x + y*WIDTH) * 4
			fog := &s.fogBuffer[x][y]
			r := min(1, fog.r*light.r*fog.hitRate+light.r*0.01)
			g := min(1, fog.g*light.g*fog.hitRate+light.g*0.01)
			b := min(1, fog.b*light.b*fog.hitRate+light.b*0.01)
			oppDen := 1 - fog.hitRate
			light.r *= oppDen * (1 - fog.hitRate + fog.r*fog.hitRate)
			light.g *= oppDen * (1 - fog.hitRate + fog.g*fog.hitRate)
			light.b *= oppDen * (1 - fog.hitRate + fog.b*fog.hitRate)
			s.frameBuffer[index] = byte(gammaCorrect(r) * 255)
			s.frameBuffer[index+1] = byte(gammaCorrect(g) * 255)
			s.frameBuffer[index+2] = byte(gammaCorrect(b) * 255)
			s.frameBuffer[index+3] = 255
		}
		// The light values get blended/blurred with a basic 1D
		// 1 radus box blur every step so it looks more natural
		for x := 0; x < WIDTH; x++ {
			s.lightBufferOut[x] = FloatColor{0, 0, 0}
			count := 0.0
			for sx := max(0, x-1); sx <= min(WIDTH-1, x+1); sx++ {
				s.lightBufferOut[x].r += s.lightBufferIn[sx].r
				s.lightBufferOut[x].g += s.lightBufferIn[sx].g
				s.lightBufferOut[x].b += s.lightBufferIn[sx].b
				count++
			}
			s.lightBufferOut[x].r /= count
			s.lightBufferOut[x].g /= count
			s.lightBufferOut[x].b /= count
		}
		// The box blurring can't really be done in-place, so we
		// need two buffers to go back and forth between
		tmp := s.lightBufferIn
		s.lightBufferIn = s.lightBufferOut
		s.lightBufferOut = tmp
	}
}

func (s *Scene) render(screen *ebiten.Image) {
	s.perCellThreaded(s.calcFog)
	s.drawFog()

	// Particles are just drawn as single red or blue pixels alpha-blended
	// onto the final image before it's written to the GPU
	if DRAW_PARTICLES {
		for _, part := range s.particles {
			if !boundsCheck(int(part.x), int(part.y)) {
				continue
			}
			index := (int(part.x) + int(part.y)*WIDTH) * 4
			var r, g, b float64 = 0, 0, 0
			if part.charge < 0 {
				r = 255
			} else {
				b = 255
			}
			s.frameBuffer[index] = uint8(float64(s.frameBuffer[index])*(1-OPACITY) + OPACITY*r)
			s.frameBuffer[index+1] = uint8(float64(s.frameBuffer[index+1])*(1-OPACITY) + OPACITY*g)
			s.frameBuffer[index+2] = uint8(float64(s.frameBuffer[index+2])*(1-OPACITY) + OPACITY*b)
		}
	}

	screen.WritePixels(s.frameBuffer)
}
