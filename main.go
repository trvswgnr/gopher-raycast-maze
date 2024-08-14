package main

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed assets/*
var assets embed.FS

const (
	screenWidth  int     = 1024
	screenHeight int     = 768
	moveSpeed    float64 = 0.05
	rotSpeed     float64 = 0.03
)

type Game struct {
	player     Player
	minimap    *ebiten.Image
	frameCount int
	level      Level
}

type Player struct {
	x, y           float64
	dirX, dirY     float64
	planeX, planeY float64
}

func NewGame() *Game {
	level := NewLevel("assets/level-1.png")
	player := Player{
		x:      level.getPlayerStartX() + 0.2,
		y:      level.getPlayerStartY() + 0.2,
		dirX:   -1,
		dirY:   0,
		planeX: 0,
		planeY: 0.66,
	}
	g := &Game{
		player:  player,
		minimap: ebiten.NewImage(level.width()*4, level.height()*4),
		level:   level,
	}
	g.generateMinimap()
	return g
}

func (level Level) getPlayerStartX() float64 {
	for y := 0; y < len(level); y++ {
		for x := 0; x < len(level[y]); x++ {
			if level[y][x] == LevelEntity_Player {
				return float64(x)
			}
		}
	}
	panic("player not found")
}

func (level Level) getPlayerStartY() float64 {
	for y := 0; y < len(level); y++ {
		for x := 0; x < len(level[y]); x++ {
			if level[y][x] == LevelEntity_Player {
				return float64(y)
			}
		}
	}
	panic("player not found")
}

func (g *Game) generateMinimap() {
	for y := 0; y < g.level.height(); y++ {
		for x := 0; x < g.level.width(); x++ {
			if g.level.getEntityAt(x, y) == LevelEntity_Wall {
				vector.DrawFilledRect(g.minimap, float32(x*4), float32(y*4), 4, 4, color.RGBA{50, 50, 50, 255}, false)
			} else {
				vector.DrawFilledRect(g.minimap, float32(x*4), float32(y*4), 4, 4, color.RGBA{140, 140, 140, 255}, false)
			}
		}
	}
}

func (g *Game) Update() error {
	g.frameCount++

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		movePlayer(g, &g.player, moveSpeed)
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		movePlayer(g, &g.player, -moveSpeed)
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		rotatePlayer(&g.player, -rotSpeed)
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		rotatePlayer(&g.player, rotSpeed)
	}

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		fmt.Println("goodbye!")
		os.Exit(0)
	}

	return nil
}

func movePlayer(g *Game, p *Player, speed float64) {
	nextX := p.x + p.dirX*speed
	nextY := p.y + p.dirY*speed

	// check if the next position is within bounds and not a wall
	if nextX >= 0 && nextX < g.level.fwidth() && g.level.getEntityAt(int(nextX), int(p.y)) != LevelEntity_Wall {
		p.x = nextX
	}
	if nextY >= 0 && nextY < g.level.fheight() && g.level.getEntityAt(int(p.x), int(nextY)) != LevelEntity_Wall {
		p.y = nextY
	}
}

func rotatePlayer(p *Player, angle float64) {
	oldDirX := p.dirX
	p.dirX = p.dirX*math.Cos(angle) - p.dirY*math.Sin(angle)
	p.dirY = oldDirX*math.Sin(angle) + p.dirY*math.Cos(angle)
	oldPlaneX := p.planeX
	p.planeX = p.planeX*math.Cos(angle) - p.planeY*math.Sin(angle)
	p.planeY = oldPlaneX*math.Sin(angle) + p.planeY*math.Cos(angle)
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	for x := 0; x < screenWidth; x++ {
		cameraX := 2*float64(x)/float64(screenWidth) - 1
		rayDirX := g.player.dirX + g.player.planeX*cameraX
		rayDirY := g.player.dirY + g.player.planeY*cameraX

		mapX, mapY := int(g.player.x), int(g.player.y)

		var sideDistX, sideDistY float64

		deltaDistX := math.Abs(1 / rayDirX)
		deltaDistY := math.Abs(1 / rayDirY)

		var stepX, stepY int
		var hit, side int

		if rayDirX < 0 {
			stepX = -1
			sideDistX = (g.player.x - float64(mapX)) * deltaDistX
		} else {
			stepX = 1
			sideDistX = (float64(mapX) + 1.0 - g.player.x) * deltaDistX
		}
		if rayDirY < 0 {
			stepY = -1
			sideDistY = (g.player.y - float64(mapY)) * deltaDistY
		} else {
			stepY = 1
			sideDistY = (float64(mapY) + 1.0 - g.player.y) * deltaDistY
		}

		for hit == 0 {
			if sideDistX < sideDistY {
				sideDistX += deltaDistX
				mapX += stepX
				side = 0
			} else {
				sideDistY += deltaDistY
				mapY += stepY
				side = 1
			}
			if g.level.getEntityAt(mapX, mapY) != LevelEntity_Empty {
				hit = 1
			}
		}

		var perpWallDist float64
		if side == 0 {
			perpWallDist = (float64(mapX) - g.player.x + (1-float64(stepX))/2) / rayDirX
		} else {
			perpWallDist = (float64(mapY) - g.player.y + (1-float64(stepY))/2) / rayDirY
		}

		lineHeight := int(float64(screenHeight) / perpWallDist)

		drawStart := -lineHeight/2 + screenHeight/2
		if drawStart < 0 {
			drawStart = 0
		}
		drawEnd := lineHeight/2 + screenHeight/2
		if drawEnd >= screenHeight {
			drawEnd = screenHeight - 1
		}

		var wallColor color.RGBA
		switch g.level[mapY][mapX] {
		case LevelEntity_Wall:
			wallColor = color.RGBA{100, 100, 100, 255} // gray
		case LevelEntity_Enemy:
			wallColor = color.RGBA{58, 231, 144, 255} // springgreen
		case LevelEntity_Exit:
			wallColor = color.RGBA{95, 158, 160, 255} // cadetblue
		case LevelEntity_Player:
			wallColor = color.RGBA{218, 165, 32, 255} // goldenrod
		default:
			wallColor = color.RGBA{200, 200, 200, 255} // white
		}

		if side == 1 {
			wallColor.R = wallColor.R / 2
			wallColor.G = wallColor.G / 2
			wallColor.B = wallColor.B / 2
		}

		vector.DrawFilledRect(screen, float32(x), float32(drawStart), 1, float32(drawEnd-drawStart), wallColor, false)
	}

	// draw minimap
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(screenWidth-g.level.width()*4-10), 10)
	screen.DrawImage(g.minimap, op)

	// draw player on minimap
	vector.DrawFilledCircle(screen, float32(screenWidth-g.level.width()*4-10+int(g.player.x*4)), float32(10+int(g.player.y*4)), 2, color.RGBA{255, 0, 0, 255}, false)

	// display fps
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f", ebiten.ActualFPS()))

	// display controls
	ebitenutil.DebugPrintAt(screen, "Controls: Arrow keys to move/rotate", 10, screenHeight-40)
	ebitenutil.DebugPrintAt(screen, "ESC to exit", 10, screenHeight-20)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("First-Person Maze Game")

	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}

type LevelEntity int

const (
	LevelEntity_Empty LevelEntity = iota
	LevelEntity_Wall
	LevelEntity_Enemy
	LevelEntity_Exit
	LevelEntity_Player
)

type Level [][]LevelEntity

func NewLevel(imagePath string) Level {
	// open image file
	file, err := assets.Open(imagePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatal(err)
	}

	// get image bounds
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	// create matrix
	matrix := make(Level, height)
	for i := range matrix {
		matrix[i] = make([]LevelEntity, width)
	}

	// fill matrix based on pixel colors
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r, g, b = r>>8, g>>8, b>>8 // convert from uint32 to uint8

			switch {
			case r == 255 && g == 255 && b == 255:
				matrix[y][x] = LevelEntity_Empty // white (empty space)
			case r == 0 && g == 0 && b == 0:
				matrix[y][x] = LevelEntity_Wall // black (wall)
			case r == 255 && g == 0 && b == 0:
				matrix[y][x] = LevelEntity_Enemy // red (enemy)
			case r == 0 && g == 255 && b == 0:
				matrix[y][x] = LevelEntity_Exit // green (exit)
			case r == 0 && g == 0 && b == 255:
				matrix[y][x] = LevelEntity_Player // blue (player)
			}
		}
	}

	return matrix
}

func (l Level) width() int {
	return len(l[0])
}

func (l Level) height() int {
	return len(l)
}

func (l Level) fwidth() float64 {
	return float64(len(l[0]))
}

func (l Level) fheight() float64 {
	return float64(len(l))
}

func (l Level) getEntityAt(x, y int) LevelEntity {
	return l[y][x]
}
