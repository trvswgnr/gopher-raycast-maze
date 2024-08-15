package main

import (
	"embed"
	"fmt"
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
	moveSpeed    float64 = 0.08
	rotSpeed     float64 = 0.07
	enemySpeed   float64 = 0.03
)

type Game struct {
	player     Player
	enemies    []Enemy
	minimap    *ebiten.Image
	frameCount int
	level      Level
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
	level := NewLevel("assets/level-1.png")
	playerX, playerY := level.getPlayer()
	player := Player{
		x:      playerX + 0.2,
		y:      playerY + 0.2,
		dirX:   -1,
		dirY:   0,
		planeX: 0,
		planeY: 0.66,
	}
	enemies := level.getEnemies()
	g := &Game{
		player:  player,
		minimap: ebiten.NewImage(level.width()*4, level.height()*4),
		level:   level,
		enemies: enemies,
	}

	g.generateStaticMinimap()

	return g
}

func (g *Game) generateStaticMinimap() {
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

	g.handleInput()

	return nil
}

func (g *Game) handleInput() {
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		movePlayer(g, &g.player, moveSpeed)
	} else if ebiten.IsKeyPressed(ebiten.KeyDown) {
		movePlayer(g, &g.player, -moveSpeed)
	}

	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		rotatePlayer(&g.player, -rotSpeed)
	} else if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		rotatePlayer(&g.player, rotSpeed)
	}

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		fmt.Println("goodbye!")
		os.Exit(0)
	}
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
	floorColor := color.RGBA{30, 30, 30, 255}      // dark gray for floor
	ceilingColor := color.RGBA{160, 227, 254, 255} // sky blue for ceiling

	for y := 0; y < screenHeight; y++ {
		if y < screenHeight/2 {
			// draw ceiling
			vector.DrawFilledRect(screen, 0, float32(y), float32(screenWidth), 1, ceilingColor, false)
		} else {
			// draw floor
			vector.DrawFilledRect(screen, 0, float32(y), float32(screenWidth), 1, floorColor, false)
		}
	}

	for x := 0; x < screenWidth; x++ {
		cameraX := 2*float64(x)/float64(screenWidth) - 1
		rayDirX := g.player.dirX + g.player.planeX*cameraX
		rayDirY := g.player.dirY + g.player.planeY*cameraX

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

		var perpWallDist float64
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
			hitEntity := g.level.getEntityAt(mapX, mapY)
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

		// render entities from far to near
		for i := len(entities) - 1; i >= 0; i-- {
			entity := entities[i]
			perpWallDist = entity.dist

			lineHeight := int(float64(screenHeight) / perpWallDist)
			drawStart := -lineHeight/2 + screenHeight/2
			drawEnd := lineHeight/2 + screenHeight/2

			// increase the height only for walls, keeping the bottom aligned
			if entity.entity == LevelEntity_Wall {
				factor := 2.0 // adjusting this value will change the height of the walls
				tallLineHeight := int(float64(lineHeight) * factor)
				drawStart = drawEnd - tallLineHeight
			}

			if drawStart < 0 {
				drawStart = 0
			}
			if drawEnd >= screenHeight {
				drawEnd = screenHeight - 1
			}

			var wallColor color.RGBA
			switch entity.entity {
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

			if entity.side == 1 {
				wallColor.R = wallColor.R / 2
				wallColor.G = wallColor.G / 2
				wallColor.B = wallColor.B / 2
			}

			vector.DrawFilledRect(screen, float32(x), float32(drawStart), 1, float32(drawEnd-drawStart), wallColor, false)
		}
	}

	// draw minimap
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(screenWidth-g.level.width()*4-10), 10)
	screen.DrawImage(g.minimap, op)

	// draw player on minimap
	vector.DrawFilledCircle(
		screen,
		float32(screenWidth-g.level.width()*4-10+int(g.player.x*4)),
		float32(10+int(g.player.y*4)),
		2,
		color.RGBA{255, 0, 0, 255},
		false,
	)

	// draw enemies on minimap
	for _, enemy := range g.enemies {
		vector.DrawFilledCircle(
			screen,
			float32(screenWidth-g.level.width()*4-10+int(enemy.x*4)),
			float32(10+int(enemy.y*4)),
			2,
			color.RGBA{0, 255, 0, 255},
			false,
		)
	}

	// display fps
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f", ebiten.ActualFPS()))

	// display controls
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
