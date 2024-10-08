package main

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	"io/fs"
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
	moveSpeed    float64 = 0.08
	rotSpeed     float64 = 0.07
	enemySpeed   float64 = 0.03
)

type Game struct {
	player  Player
	enemies []Enemy
	minimap *ebiten.Image
	level   Level
}

type Enemy struct {
	x, y float64
	// dirx, diry float64
}

type Player struct {
	x, y           float64
	dirX, dirY     float64
	planeX, planeY float64
}

func NewGame() *Game {
	file, err := assets.Open("assets/level-1.png")
	if err != nil {
		log.Fatal(err)
	}
	level := NewLevel(file)
	playerX, playerY := level.GetPlayer()
	player := Player{
		x:      playerX + 0.2,
		y:      playerY + 0.2,
		dirX:   -1,
		dirY:   0,
		planeX: 0,
		planeY: 0.66,
	}
	enemies := level.GetEnemies()
	g := &Game{
		player:  player,
		minimap: ebiten.NewImage(level.Width()*4, level.Height()*4),
		level:   level,
		enemies: enemies,
	}

	g.generateStaticMinimap()

	return g
}

func (g *Game) generateStaticMinimap() {
	for y := 0; y < g.level.Height(); y++ {
		for x := 0; x < g.level.Width(); x++ {
			if g.level.GetEntityAt(x, y) == LevelEntity_Wall {
				vector.DrawFilledRect(g.minimap, float32(x*4), float32(y*4), 4, 4, color.RGBA{50, 50, 50, 255}, false)
			} else {
				vector.DrawFilledRect(g.minimap, float32(x*4), float32(y*4), 4, 4, color.RGBA{140, 140, 140, 255}, false)
			}
		}
	}
}

func (g *Game) Update() error {
	g.handleInput()

	return nil
}

func (g *Game) handleInput() {
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.movePlayer(moveSpeed)
	} else if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.movePlayer(-moveSpeed)
	}

	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.rotatePlayer(-rotSpeed)
	} else if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.rotatePlayer(rotSpeed)
	}

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		os.Exit(0)
	}
}

func (g *Game) movePlayer(speed float64) {
	nextX := g.player.x + g.player.dirX*speed
	nextY := g.player.y + g.player.dirY*speed

	// Check collision with walls and enemies
	if !g.playerCollision(nextX, g.player.y) {
		g.player.x = nextX
	}
	if !g.playerCollision(g.player.x, nextY) {
		g.player.y = nextY
	}
}

func (g *Game) playerCollision(x, y float64) bool {
	// check if the position is within the level bounds
	if x < 0 || y < 0 || int(x) >= g.level.Width() || int(y) >= g.level.Height() {
		return true
	}

	// check if the position is a wall
	if g.level.GetEntityAt(int(x), int(y)) == LevelEntity_Wall {
		return true
	}

	// check if the position is an enemy
	if g.level.GetEntityAt(int(x), int(y)) == LevelEntity_Enemy {
		return true
	}

	return false
}

func (g *Game) rotatePlayer(angle float64) {
	oldDirX := g.player.dirX
	g.player.dirX = g.player.dirX*math.Cos(angle) - g.player.dirY*math.Sin(angle)
	g.player.dirY = oldDirX*math.Sin(angle) + g.player.dirY*math.Cos(angle)
	oldPlaneX := g.player.planeX
	g.player.planeX = g.player.planeX*math.Cos(angle) - g.player.planeY*math.Sin(angle)
	g.player.planeY = oldPlaneX*math.Sin(angle) + g.player.planeY*math.Cos(angle)
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.drawFloorAndCeiling(screen)
	g.drawBlocks(screen)
	g.drawMinimap(screen)
	g.drawUI(screen)
}

func (g *Game) drawFloorAndCeiling(screen *ebiten.Image) {
	floorColor := color.RGBA{30, 30, 30, 255}
	ceilingColor := color.RGBA{160, 227, 254, 255}

	for y := 0; y < screenHeight; y++ {
		if y < screenHeight/2 {
			vector.DrawFilledRect(screen, 0, float32(y), float32(screenWidth), 1, ceilingColor, false)
		} else {
			vector.DrawFilledRect(screen, 0, float32(y), float32(screenWidth), 1, floorColor, false)
		}
	}
}

func (g *Game) drawBlocks(screen *ebiten.Image) {
	for x := 0; x < screenWidth; x++ {
		rayDirX, rayDirY := g.calculateRayDirection(x)
		entities := g.castRay(rayDirX, rayDirY)
		g.drawEntities(screen, x, entities)
	}
}

func (g *Game) calculateRayDirection(x int) (float64, float64) {
	cameraX := 2*float64(x)/float64(screenWidth) - 1
	rayDirX := g.player.dirX + g.player.planeX*cameraX
	rayDirY := g.player.dirY + g.player.planeY*cameraX
	return rayDirX, rayDirY
}

func (g *Game) castRay(rayDirX, rayDirY float64) []struct {
	entity LevelEntity
	dist   float64
	side   int
} {
	mapX, mapY := int(g.player.x), int(g.player.y)
	var sideDistX, sideDistY float64
	deltaDistX := math.Abs(1 / rayDirX)
	deltaDistY := math.Abs(1 / rayDirY)
	var stepX, stepY int
	var side int

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

	var hitWall bool
	var entities []struct {
		entity LevelEntity
		dist   float64
		side   int
	}

	for !hitWall {
		if sideDistX < sideDistY {
			sideDistX += deltaDistX
			mapX += stepX
			side = 0
		} else {
			sideDistY += deltaDistY
			mapY += stepY
			side = 1
		}
		hitEntity := g.level.GetEntityAt(mapX, mapY)
		if hitEntity != LevelEntity_Empty {
			var dist float64
			if side == 0 {
				dist = (float64(mapX) - g.player.x + (1-float64(stepX))/2) / rayDirX
			} else {
				dist = (float64(mapY) - g.player.y + (1-float64(stepY))/2) / rayDirY
			}
			entities = append(entities, struct {
				entity LevelEntity
				dist   float64
				side   int
			}{hitEntity, dist, side})

			if hitEntity == LevelEntity_Wall {
				hitWall = true
			}
		}
	}

	return entities
}

func (g *Game) drawEntities(screen *ebiten.Image, x int, entities []struct {
	entity LevelEntity
	dist   float64
	side   int
}) {
	for i := len(entities) - 1; i >= 0; i-- {
		entity := entities[i]
		_, drawStart, drawEnd := g.calculateLineParameters(entity.dist, entity.entity)
		wallColor := g.getEntityColor(entity.entity, entity.side)
		vector.DrawFilledRect(screen, float32(x), float32(drawStart), 1, float32(drawEnd-drawStart), wallColor, false)
	}
}

func (g *Game) calculateLineParameters(dist float64, entity LevelEntity) (int, int, int) {
	lineHeight := int(float64(screenHeight) / dist)
	drawStart := -lineHeight/2 + screenHeight/2
	drawEnd := lineHeight/2 + screenHeight/2

	if entity == LevelEntity_Wall {
		factor := 2.0
		lineHeight = int(float64(lineHeight) * factor)
		drawStart = drawEnd - lineHeight
	}

	if drawStart < 0 {
		drawStart = 0
	}
	if drawEnd >= screenHeight {
		drawEnd = screenHeight - 1
	}

	return lineHeight, drawStart, drawEnd
}

func (g *Game) getEntityColor(entity LevelEntity, side int) color.RGBA {
	var entityColor color.RGBA
	switch entity {
	case LevelEntity_Wall:
		entityColor = color.RGBA{100, 100, 100, 255}
	case LevelEntity_Enemy:
		entityColor = color.RGBA{58, 231, 144, 255}
	case LevelEntity_Exit:
		entityColor = color.RGBA{95, 158, 160, 255}
	case LevelEntity_Player:
		entityColor = color.RGBA{218, 165, 32, 255}
	default:
		entityColor = color.RGBA{200, 200, 200, 255}
	}

	if side == 1 {
		entityColor.R = entityColor.R / 2
		entityColor.G = entityColor.G / 2
		entityColor.B = entityColor.B / 2
	}

	return entityColor
}

func (g *Game) drawMinimap(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(screenWidth-g.level.Width()*4-10), 10)
	screen.DrawImage(g.minimap, op)

	g.drawPlayerOnMinimap(screen)
	g.drawEnemiesOnMinimap(screen)
}

func (g *Game) drawPlayerOnMinimap(screen *ebiten.Image) {
	vector.DrawFilledCircle(
		screen,
		float32(screenWidth-g.level.Width()*4-10+int(g.player.x*4)),
		float32(10+int(g.player.y*4)),
		2,
		color.RGBA{255, 0, 0, 255},
		false,
	)
}

func (g *Game) drawEnemiesOnMinimap(screen *ebiten.Image) {
	for _, enemy := range g.enemies {
		vector.DrawFilledCircle(
			screen,
			float32(screenWidth-g.level.Width()*4-10+int(enemy.x*4)),
			float32(10+int(enemy.y*4)),
			2,
			color.RGBA{0, 255, 0, 255},
			false,
		)
	}
}

func (g *Game) drawUI(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("FPS: %0.2f", ebiten.ActualFPS()), 10, 10)
	ebitenutil.DebugPrintAt(screen, "move with arrow keys", 10, screenHeight-40)
	ebitenutil.DebugPrintAt(screen, "ESC to exit", 10, screenHeight-20)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("maze 3d raycasting")

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

func NewLevel(file fs.File) Level {
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatal(err)
	}

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

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
			case r == 255 && g == 255 && b == 255: // white (empty space)
				matrix[y][x] = LevelEntity_Empty
			case r == 0 && g == 0 && b == 0: // black (wall)
				matrix[y][x] = LevelEntity_Wall
			case r == 255 && g == 0 && b == 0: // red (enemy)
				matrix[y][x] = LevelEntity_Enemy
			case r == 0 && g == 255 && b == 0: // green (exit)
				matrix[y][x] = LevelEntity_Exit
			case r == 0 && g == 0 && b == 255: // blue (player)
				matrix[y][x] = LevelEntity_Player
			}
		}
	}

	return matrix
}

func (level Level) GetPlayer() (float64, float64) {
	playerX := 0
	playerY := 0
	for y := 0; y < len(level); y++ {
		for x := 0; x < len(level[y]); x++ {
			if level[y][x] == LevelEntity_Player {
				playerX = x
			}
		}
	}

	for y := 0; y < len(level); y++ {
		for x := 0; x < len(level[y]); x++ {
			if level[y][x] == LevelEntity_Player {
				playerY = y
			}
		}
	}

	// remove player from level
	level[playerY][playerX] = LevelEntity_Empty

	return float64(playerX), float64(playerY)
}

func (level Level) GetEnemies() []Enemy {
	enemies := []Enemy{}
	for y := 0; y < len(level); y++ {
		for x := 0; x < len(level[y]); x++ {
			if level[y][x] == LevelEntity_Enemy {
				enemies = append(enemies, Enemy{x: float64(x), y: float64(y)})
			}
		}
	}
	return enemies
}

func (l Level) Width() int {
	return len(l[0])
}

func (l Level) Height() int {
	return len(l)
}

func (l Level) Fwidth() float64 {
	return float64(len(l[0]))
}

func (l Level) Fheight() float64 {
	return float64(len(l))
}

func (l Level) GetEntityAt(x, y int) LevelEntity {
	return l[y][x]
}
