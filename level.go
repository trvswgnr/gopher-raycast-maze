package main

import (
	"image"
	"log"
)

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

func (level Level) getPlayer() (float64, float64) {
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

	return float64(playerX), float64(playerY)
}

func (level Level) getEnemies() []Enemy {
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