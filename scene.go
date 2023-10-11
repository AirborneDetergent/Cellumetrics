package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Scene struct {
	frameBuffer    []byte
	fogBuffer      [WIDTH][HEIGHT]FogCell
	cellsIn        *[WIDTH][HEIGHT]Cell
	cellsOut       *[WIDTH][HEIGHT]Cell
	particles      [PARTICLE_COUNT]Particle
	lightBufferIn  *[WIDTH]FloatColor
	lightBufferOut *[WIDTH]FloatColor
}

func newScene() *Scene {
	s := &Scene{}
	s.frameBuffer = make([]byte, WIDTH*HEIGHT*4)
	s.cellsIn = &[WIDTH][HEIGHT]Cell{}
	s.cellsOut = &[WIDTH][HEIGHT]Cell{}
	s.lightBufferIn = &[WIDTH]FloatColor{}
	s.lightBufferOut = &[WIDTH]FloatColor{}
	for i := range s.particles {
		p := &s.particles[i]
		p.x = rand.Float64() * WIDTH
		p.y = rand.Float64() * HEIGHT
		if rand.Float64() < NEGATIVE_RATE {
			p.charge = -1
		} else {
			p.charge = 1
		}
	}
	return s
}

func (s *Scene) Update() error {
	return nil
}

func (s *Scene) Draw(screen *ebiten.Image) {
	debugInfo := ""
	timer := makeTimer()
	s.simStep()
	simTime = simTime*0.9 + timer.tick()*0.1
	s.render(screen)
	renderTime = renderTime*0.9 + timer.tick()*0.1
	debugInfo += fmt.Sprintf("FPS: %0.4g\n", ebiten.ActualFPS())
	debugInfo += fmt.Sprintf("Simulation time: %0.3f\n", simTime)
	debugInfo += fmt.Sprintf("Render time: %0.3f\n", renderTime)
	ebitenutil.DebugPrint(screen, debugInfo)
}

func (s *Scene) readChargeBounded(x, y int) float64 {
	if boundsCheck(x, y) {
		return s.cellsIn[x][y].charge
	}
	return 0.0
}

func (s *Scene) approxDeriv(x, y int) (dx, dy float64) {
	dx = s.readChargeBounded(x+1, y) - s.readChargeBounded(x-1, y)
	dy = s.readChargeBounded(x, y+1) - s.readChargeBounded(x, y-1)
	return dx, dy
}

func (s *Scene) perCellThreaded(fun func(int, int)) {
	wg := sync.WaitGroup{}
	wg.Add(THREADS)
	for i := 0; i < THREADS; i++ {
		go s.perCellThreadedHelper(fun, i, &wg)
	}
	wg.Wait()
}

func (s *Scene) perCellThreadedHelper(fun func(int, int), offset int, wg *sync.WaitGroup) {
	for x := offset; x < WIDTH; x += THREADS {
		for y := range s.cellsIn[0] {
			fun(x, y)
		}
	}
	wg.Done()
}

func (s *Scene) diffusion(x, y int) {
	s.cellsOut[x][y].charge = 0
	count := 0
	for sx := x - 1; sx <= x+1; sx++ {
		for sy := y - 1; sy <= y+1; sy++ {
			if sx >= 0 && sx < WIDTH && sy >= 0 && sy < HEIGHT {
				count++
				s.cellsOut[x][y].charge += s.cellsIn[sx][sy].charge
			}
		}
	}
	s.cellsOut[x][y].charge /= float64(count)
	s.cellsOut[x][y].charge *= 0.999
}

func (s *Scene) simStep() {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		for i := 0; i < 100; i++ {
			mouseX, mouseY := ebiten.CursorPosition()
			mouseX += rand.Intn(32) - 16
			mouseY += rand.Intn(32) - 16
			if !boundsCheck(mouseX, mouseY) {
				return
			}
			s.cellsIn[mouseX][mouseY].charge += 0.1
		}

	}

	s.perCellThreaded(s.diffusion)

	for i := range s.particles {
		p := &s.particles[i]
		dx, dy := s.approxDeriv(int(p.x), int(p.y))
		p.xv -= dx / 144 * p.charge
		p.yv -= dy / 144 * p.charge
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
		p.xv *= 0.995
		p.yv *= 0.995
		p.x += p.xv
		p.y += p.yv
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

	for i := range s.particles {
		p := &s.particles[i]
		if boundsCheck(int(p.x), int(p.y)) {
			s.cellsOut[int(p.x)][int(p.y)].charge += EMISSION_STRENGTH * p.charge
		}
	}

	tmp := s.cellsIn
	s.cellsIn = s.cellsOut
	s.cellsOut = tmp
}

func (s *Scene) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return WIDTH, HEIGHT
}

func (s *Scene) calcFog(x, y int) {
	cell := &s.cellsIn[x][y]
	charge := cell.charge / MAX_CHARGE_RENDERED
	fog := FogCell{}
	fog.r, fog.g, fog.b = getColor(charge * 10)
	fog.hitRate = 1 - math.Pow(0.5, math.Abs(charge))
	s.fogBuffer[x][y] = fog
}

// Muh CPU can't handle 2.2
func gammaCorrect(n float64) float64 {
	return n * n
}

func (s *Scene) drawFog() {
	for i := range s.lightBufferIn {
		s.lightBufferIn[i] = FloatColor{50, 50, 50}
	}
	for y := 0; y < HEIGHT; y++ {
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
		for x := 0; x < WIDTH; x++ {
			count := 0.0
			s.lightBufferOut[x] = FloatColor{0, 0, 0}
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
		tmp := s.lightBufferIn
		s.lightBufferIn = s.lightBufferOut
		s.lightBufferOut = tmp
	}
}

func (s *Scene) render(screen *ebiten.Image) {
	s.perCellThreaded(s.calcFog)

	//wg := sync.WaitGroup{}
	//wg.Add(THREADS)
	//for i := 0; i < THREADS; i++ {
	//	go s.drawFog(i, &wg)
	//}
	//wg.Wait()
	s.drawFog()

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
