package captcha

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

const (
	// 验证码长度
	CodeLength = 4
	// 验证码有效期
	Expiry = 5 * time.Minute
	// 最大存储数
	MaxStore = 10000
	// 字符集（去掉容易混淆的字符）
	Charset = "2345678ABCDEFGHJKLMNPQRSTUVWXYZ"
)

type captchaEntry struct {
	Code      string
	ExpiresAt time.Time
}

var (
	store = make(map[string]*captchaEntry)
	mu    sync.RWMutex
)

// Generate 生成验证码，返回 (captchaID, svgData)
func Generate() (string, string) {
	// 清理过期
	cleanExpired()

	id := randomString(32)
	code := randomCode(CodeLength)

	mu.Lock()
	store[id] = &captchaEntry{
		Code:      code,
		ExpiresAt: time.Now().Add(Expiry),
	}
	mu.Unlock()

	svg := renderSVG(code)
	return id, svg
}

// Verify 验证验证码，验证后立即删除（一次性）
func Verify(id, code string) bool {
	if id == "" || code == "" {
		return false
	}

	mu.Lock()
	defer mu.Unlock()

	entry, ok := store[id]
	if !ok {
		return false
	}

	// 删除（一次性使用）
	delete(store, id)

	if time.Now().After(entry.ExpiresAt) {
		return false
	}

	return strings.EqualFold(entry.Code, code)
}

func cleanExpired() {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	for k, v := range store {
		if now.After(v.ExpiresAt) {
			delete(store, k)
		}
	}

	// 防止内存溢出：如果超过上限，全部清除
	if len(store) > MaxStore {
		store = make(map[string]*captchaEntry)
	}
}

func randomCode(length int) string {
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(Charset))))
		result[i] = Charset[n.Int64()]
	}
	return string(result)
}

func randomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[n.Int64()]
	}
	return string(result)
}

// renderSVG 生成带干扰的 SVG 验证码图片
func renderSVG(code string) string {
	width := 150
	height := 50

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, width, height, width, height))
	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" fill="#f0f0f0"/>`, width, height))

	// 干扰线
	for i := 0; i < 6; i++ {
		x1 := randInt(0, width)
		y1 := randInt(0, height)
		x2 := randInt(0, width)
		y2 := randInt(0, height)
		color := randColor()
		sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1" opacity="0.5"/>`, x1, y1, x2, y2, color))
	}

	// 干扰点
	for i := 0; i < 30; i++ {
		x := randInt(0, width)
		y := randInt(0, height)
		color := randColor()
		sb.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="1" fill="%s" opacity="0.5"/>`, x, y, color))
	}

	// 干扰曲线
	for i := 0; i < 2; i++ {
		startX := randInt(0, 20)
		startY := randInt(10, 40)
		cp1x := randInt(30, 60)
		cp1y := randInt(0, 50)
		cp2x := randInt(80, 120)
		cp2y := randInt(0, 50)
		endX := randInt(130, 150)
		endY := randInt(10, 40)
		color := randColor()
		sb.WriteString(fmt.Sprintf(`<path d="M%d,%d C%d,%d %d,%d %d,%d" fill="none" stroke="%s" stroke-width="1.5" opacity="0.6"/>`,
			startX, startY, cp1x, cp1y, cp2x, cp2y, endX, endY, color))
	}

	// 字符
	charWidth := width / (len(code) + 1)
	for i, ch := range code {
		x := charWidth*(i+1) - 5
		y := randInt(28, 38)
		fontSize := randInt(22, 30)
		rotate := randInt(-20, 20)
		color := randDarkColor()
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-size="%d" font-family="Arial,sans-serif" font-weight="bold" fill="%s" transform="rotate(%d,%d,%d)">%c</text>`,
			x, y, fontSize, color, rotate, x, y, ch))
	}

	sb.WriteString(`</svg>`)
	return sb.String()
}

func randInt(min, max int) int {
	if min >= max {
		return min
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min)))
	return int(n.Int64()) + min
}

func randColor() string {
	r := randInt(80, 220)
	g := randInt(80, 220)
	b := randInt(80, 220)
	return fmt.Sprintf("rgb(%d,%d,%d)", r, g, b)
}

func randDarkColor() string {
	r := randInt(20, 120)
	g := randInt(20, 120)
	b := randInt(20, 120)
	return fmt.Sprintf("rgb(%d,%d,%d)", r, g, b)
}
