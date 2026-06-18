package handlers

import (
	"fmt"
	"html/template"
	"math"
	"strconv"
	"strings"
)

func normalizeBoardColor(color string) string {
	color = strings.TrimSpace(color)
	if isHexColor(color) {
		return strings.ToLower(color)
	}
	return boardNamedColorHex(color)
}

func boardHexColor(color string) string {
	color = strings.TrimSpace(color)
	if isHexColor(color) {
		return strings.ToLower(color)
	}
	return boardNamedColorHex(color)
}

func boardHeaderStyle(color string) template.CSS {
	hex := boardHexColor(color)
	text := "#ffffff"
	if colorLuminance(hex) > 0.55 {
		text = "#111111"
	}
	return template.CSS(fmt.Sprintf("background-color:%s;color:%s", hex, text))
}

func boardNamedColorHex(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "danger":
		return "#dc3545"
	case "warning":
		return "#ffc107"
	case "info":
		return "#0dcaf0"
	case "success":
		return "#198754"
	case "secondary":
		return "#6c757d"
	default:
		return "#0d6efd"
	}
}

func isHexColor(color string) bool {
	if len(color) != 7 || color[0] != '#' {
		return false
	}
	_, err := strconv.ParseUint(color[1:], 16, 24)
	return err == nil
}

func colorLuminance(hex string) float64 {
	r, g, b := hexToRGB(hex)
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func hexToRGB(hex string) (float64, float64, float64) {
	n, err := strconv.ParseUint(hex[1:], 16, 24)
	if err != nil {
		return 0.05, 0.4, 0.9
	}
	r := float64((n>>16)&0xff) / 255.0
	g := float64((n>>8)&0xff) / 255.0
	b := float64(n&0xff) / 255.0

	convert := func(c float64) float64 {
		if c <= 0.03928 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return convert(r), convert(g), convert(b)
}
