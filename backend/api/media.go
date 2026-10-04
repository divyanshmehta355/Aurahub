package api

import (
	"fmt"
	"io"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func fnv1a(str string) uint32 {
	var hash uint32 = 2166136261
	for i := 0; i < len(str); i++ {
		hash ^= uint32(str[i])
		hash += (hash << 1) + (hash << 4) + (hash << 7) + (hash << 8) + (hash << 24)
	}
	return hash
}

func generateIdenticonSvg(seed string) string {
	cleanSeed := strings.ToLower(strings.TrimSpace(seed))
	if cleanSeed == "" {
		cleanSeed = "user"
	}

	h1 := fnv1a(cleanSeed)
	h2 := fnv1a(cleanSeed + "_prime")
	h3 := fnv1a(cleanSeed + "_seed")
	h4 := fnv1a(cleanSeed + "_identicon")

	var bytes []byte
	for i := 0; i < 4; i++ {
		bytes = append(bytes, byte((h1>>(i*8))&0xff))
	}
	for i := 0; i < 4; i++ {
		bytes = append(bytes, byte((h2>>(i*8))&0xff))
	}
	for i := 0; i < 4; i++ {
		bytes = append(bytes, byte((h3>>(i*8))&0xff))
	}
	for i := 0; i < 4; i++ {
		bytes = append(bytes, byte((h4>>(i*8))&0xff))
	}

	hue := (uint32(bytes[0])<<8 | uint32(bytes[1])) % 360
	saturation := 65 + (int(bytes[2]) % 20)
	lightness := 44 + (int(bytes[3]) % 14)

	fgColor := fmt.Sprintf("hsl(%d, %d%%, %d%%)", hue, saturation, lightness)
	bgColor := "#f1f3f5"

	cellSize := 70
	margin := 35
	size := margin*2 + cellSize*5

	var rects strings.Builder
	filledCount := 0

	for row := 0; row < 5; row++ {
		for col := 0; col < 3; col++ {
			idx := row*3 + col
			isFilled := (int(bytes[idx%len(bytes)])+idx)%2 == 0

			if isFilled {
				filledCount++
				y := margin + row*cellSize

				x1 := margin + col*cellSize
				rects.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`, x1, y, cellSize, cellSize, fgColor))

				if col < 2 {
					x2 := margin + (4-col)*cellSize
					rects.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`, x2, y, cellSize, cellSize, fgColor))
				}
			}
		}
	}

	if filledCount == 0 {
		yCenter := margin + 2*cellSize
		xCenter := margin + 2*cellSize
		rects.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`, xCenter, yCenter, cellSize, cellSize, fgColor))
	}

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %[1]d %[1]d" width="100%%" height="100%%">
  <rect width="%[1]d" height="%[1]d" fill="%[2]s" rx="60"/>
  %[3]s
</svg>`, size, bgColor, rects.String())
}

func (s *Server) AvatarHandler(c *fiber.Ctx) error {
	seed := c.Params("seed")
	svg := generateIdenticonSvg(seed)

	c.Set("Content-Type", "image/svg+xml; charset=utf-8")
	c.Set("Cache-Control", "public, max-age=31536000, immutable")
	return c.SendString(svg)
}

func (s *Server) AvatarUploadHandler(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid form"})
	}

	files := form.File["avatar"]
	if len(files) == 0 {
		return c.Status(400).JSON(fiber.Map{"message": "No file uploaded"})
	}

	file, err := files[0].Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to read file"})
	}
	defer file.Close()

	imageData, err := io.ReadAll(io.LimitReader(file, 10<<20+1))
	if err != nil || len(imageData) > 10<<20 {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid avatar file"})
	}
	imageURL, err := uploadImage(imageData)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"message": "Failed to upload to host"})
	}
	return c.JSON(fiber.Map{"url": imageURL})
}
