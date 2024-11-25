//go:generate tailwindcss -i styles/main.css -o assets/tailwind.css --minify

package main

import (
	"bytes"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/fs"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chai2010/webp"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/handlebars/v2"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/juls0730/passport/middleware"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nfnt/resize"
)

//go:embed assets/** templates/** schema.sql
var embeddedAssets embed.FS

var devContent string = `<script>
let host = window.location.hostname;
const socket = new WebSocket('ws://' + host + ':2067/ws'); 

socket.addEventListener('message', (event) => {
    if (event.data === 'refresh') {
        async function testPage() {
            try {
            let res = await fetch(window.location.href)
            } catch (error) {
                console.error(error);
                setTimeout(testPage, 300);
                return;
            }
            window.location.reload();
        }

        testPage();
    }
});
</script>`

type Category struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Links []Link `json:"links"`
}

type Link struct {
	ID          int64  `json:"id"`
	CategoryID  int64  `json:"category_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	URL         string `json:"url"`
}

type App struct {
	db *sql.DB
	*WeatherCache
}

func NewApp(dbPath string) (*App, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	schema, err := embeddedAssets.ReadFile("schema.sql")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(string(schema))

	return &App{
		db:           db,
		WeatherCache: NewWeatherCache(),
	}, nil
}

func (app *App) GetCategories() ([]Category, error) {
	rows, err := app.db.Query(`
		SELECT id, name, icon
		FROM categories 
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Icon); err != nil {
			return nil, err
		}

		links, err := app.GetLinksByCategory(cat.ID)
		if err != nil {
			return nil, err
		}
		cat.Links = links
		categories = append(categories, cat)
	}
	return categories, nil
}

func (app *App) GetLinksByCategory(categoryID int64) ([]Link, error) {
	rows, err := app.db.Query(`
		SELECT id, category_id, name, description, icon, url 
		FROM links 
		WHERE category_id = ? 
		ORDER BY id ASC
	`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var link Link
		if err := rows.Scan(&link.ID, &link.CategoryID, &link.Name, &link.Description,
			&link.Icon, &link.URL); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

type OpenWeatherResponse struct {
	Weather []struct {
		Name   string `json:"main"`
		IconId string `json:"icon"`
	} `json:"weather"`
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
	Code    int    `json:"cod"`
	Message string `json:"message"`
}

type WeatherData struct {
	Temperature float64
	WeatherText string
	Icon        string
}

type WeatherCache struct {
	data           *WeatherData
	lastUpdate     time.Time
	mutex          sync.RWMutex
	updateChan     chan struct{}
	tempUnits      string
	updateInterval int
	apiKey         string
	lat            string
	lon            string
}

func NewWeatherCache() *WeatherCache {
	if os.Getenv("OPENWEATHER_API_KEY") == "" || os.Getenv("OPENWEATHER_LAT") == "" || os.Getenv("OPENWEATHER_LON") == "" {
		log.Fatalln("OpenWeather API Key, and your latitude and longitude are required!")
		return nil
	}

	updateInterval, err := strconv.Atoi(os.Getenv("OPENWEATHER_UPDATE_INTERVAL"))
	if err != nil || updateInterval < 1 {
		updateInterval = 15
	}

	units := os.Getenv("OPENWEATHER_TEMP_UNITS")
	if units == "" {
		units = "metric"
	}

	cache := &WeatherCache{
		data:           &WeatherData{},
		updateChan:     make(chan struct{}),
		tempUnits:      units,
		updateInterval: updateInterval,
		apiKey:         os.Getenv("OPENWEATHER_API_KEY"),
		lat:            os.Getenv("OPENWEATHER_LAT"),
		lon:            os.Getenv("OPENWEATHER_LON"),
	}

	go cache.weatherWorker()

	cache.updateChan <- struct{}{}

	return cache
}

func (c *WeatherCache) GetWeather() WeatherData {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return *c.data
}

func (c *WeatherCache) weatherWorker() {
	ticker := time.NewTicker(time.Duration(c.updateInterval) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.updateChan:
			c.updateWeather()
		case <-ticker.C:
			c.updateWeather()
		}
	}
}

func (c *WeatherCache) updateWeather() {
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%s&lon=%s&appid=%s&units=%s",
		c.lat, c.lon, c.apiKey, c.tempUnits)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error fetching weather: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	var weatherResp OpenWeatherResponse
	if err := json.Unmarshal(body, &weatherResp); err != nil {
		fmt.Printf("Error parsing weather data: %v\n", err)
		return
	}

	// if the request failed
	if weatherResp.Code != 200 {
		// if there is no pre-existing data in the cache
		if c.data.WeatherText == "" {
			log.Fatalf("Fetching the weather data failed!\n%s\n", weatherResp.Message)
		} else {
			return
		}
	}

	c.mutex.Lock()
	c.data.Temperature = weatherResp.Main.Temp
	c.data.WeatherText = weatherResp.Weather[0].Name
	c.data.Icon = weatherResp.Weather[0].IconId
	c.lastUpdate = time.Now()
	c.mutex.Unlock()
}

type CreateLinkRequest struct {
	Name        string                `form:"name"`
	Description string                `form:"description"`
	URL         string                `form:"url"`
	Icon        *multipart.FileHeader `form:"icon"`
	CategoryID  int64                 `form:"category_id"`
}

func UploadFile(file *multipart.FileHeader, fileName, contentType string, c fiber.Ctx) (string, error) {
	srcFile, err := file.Open()
	if err != nil {
		return "", err
	}
	defer srcFile.Close()

	var img image.Image
	switch contentType {
	case "image/jpeg":
		img, err = jpeg.Decode(srcFile)
	case "image/png":
		img, err = png.Decode(srcFile)
	case "image/webp":
		img, err = webp.Decode(srcFile)
	case "image/svg+xml":
	default:
		return "", errors.New("unsupported file type")
	}

	if err != nil {
		return "", err
	}

	assetsDir := "public/uploads"

	iconPath := filepath.Join(assetsDir, fileName)

	if contentType == "image/svg+xml" {
		if err = c.SaveFile(file, iconPath); err != nil {
			return "", err
		}
	} else {
		outFile, err := os.Create(iconPath)
		if err != nil {
			return "", err
		}
		defer outFile.Close()

		resizedImg := resize.Resize(64, 0, img, resize.MitchellNetravali)

		var buf bytes.Buffer
		options := &webp.Options{Lossless: true, Quality: 80}
		if err := webp.Encode(&buf, resizedImg, options); err != nil {
			return "", err
		}

		if _, err := io.Copy(outFile, &buf); err != nil {
			return "", err
		}
	}

	iconPath = "/uploads/" + fileName

	return iconPath, nil
}

func CreateLink(db *sql.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req CreateLinkRequest
		if err := c.Bind().MultipartForm(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Failed to parse request",
			})
		}

		if req.Name == "" || req.URL == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Name and URL are required",
			})
		}

		file, err := c.FormFile("icon")
		if err != nil || file == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Icon is required",
			})
		}

		if file.Size > 5*1024*1024 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "File size too large. Maximum size is 5MB",
			})
		}

		contentType := file.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "image/") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Only image files are allowed",
			})
		}

		filename := fmt.Sprintf("%d_%s.webp", time.Now().Unix(), strings.ReplaceAll(req.Name, " ", "_"))

		if contentType == "image/svg+xml" {
			filename = fmt.Sprintf("%d_%s.svg", time.Now().Unix(), strings.ReplaceAll(req.Name, " ", "_"))
		}

		iconPath, err := UploadFile(file, filename, contentType, c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to upload file, please try again!",
			})
		}

		link := Link{
			Name:        req.Name,
			Description: req.Description,
			URL:         req.URL,
			Icon:        iconPath,
			CategoryID:  req.CategoryID,
		}

		_, err = db.Exec(`
			INSERT INTO links (category_id, name, description, icon, url) 
			VALUES (?, ?, ?, ?, ?)`,
			link.CategoryID, link.Name, link.Description, link.Icon, link.URL)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to create link",
			})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "Link created successfully",
			"link":    link,
		})
	}
}

type CreateCategoryRequest struct {
	Name string                `form:"name"`
	Icon *multipart.FileHeader `form:"icon"`
}

func CreateCategory(db *sql.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req CreateCategoryRequest
		if err := c.Bind().MultipartForm(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Failed to parse request",
			})
		}

		if req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Name is required",
			})
		}

		file, err := c.FormFile("icon")
		if err != nil || file == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Icon is required",
			})
		}

		if file.Size > 5*1024*1024 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "File size too large. Maximum size is 5MB",
			})
		}

		contentType := file.Header.Get("Content-Type")
		if contentType != "image/svg+xml" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Only SVGs are supported for category icons!",
			})
		}

		filename := fmt.Sprintf("%d_%s.svg", time.Now().Unix(), strings.ReplaceAll(req.Name, " ", "_"))

		iconPath, err := UploadFile(file, filename, contentType, c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to upload file, please try again!",
			})
		}

		category := Category{
			Name:  req.Name,
			Icon:  iconPath,
			Links: []Link{},
		}

		_, err = db.Exec(`
			INSERT INTO categories (name, icon) 
			VALUES (?, ?)`,
			category.Name, category.Icon)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to create category",
			})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message":  "Category created successfully",
			"category": category,
		})
	}
}

var WeatherIcons = map[string]string{
	"clear-day":           `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32"><path fill="currentColor" d="M16 12.005a4 4 0 1 1-4 4a4.005 4.005 0 0 1 4-4m0-2a6 6 0 1 0 6 6a6 6 0 0 0-6-6M5.394 6.813L6.81 5.399l3.505 3.506L8.9 10.319zM2 15.005h5v2H2zm3.394 10.193L8.9 21.692l1.414 1.414l-3.505 3.506zM15 25.005h2v5h-2zm6.687-1.9l1.414-1.414l3.506 3.506l-1.414 1.414zm3.313-8.1h5v2h-5zm-3.313-6.101l3.506-3.506l1.414 1.414l-3.506 3.506zM15 2.005h2v5h-2z"/></svg>`,
	"clear-night":         `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32"><path fill="currentColor" d="M13.503 5.414a15.076 15.076 0 0 0 11.593 18.194a11.1 11.1 0 0 1-7.975 3.39c-.138 0-.278.005-.418 0a11.094 11.094 0 0 1-3.2-21.584M14.98 3a1 1 0 0 0-.175.016a13.096 13.096 0 0 0 1.825 25.981c.164.006.328 0 .49 0a13.07 13.07 0 0 0 10.703-5.555a1.01 1.01 0 0 0-.783-1.565A13.08 13.08 0 0 1 15.89 4.38A1.015 1.015 0 0 0 14.98 3"/></svg>`,
	"partly-cloudy-day":   `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32"><path fill="currentColor" d="M27 15h4v2h-4zm-4-7.413l3-3L27.415 6l-3 3zM15 1h2v4h-2zM4.586 26l3-3l1.415 1.415l-3 3zM4.585 6L6 4.587l3 3l-1.414 1.415z"/><path fill="currentColor" d="M1 15h4v2H1zm25.794 5.342a6.96 6.96 0 0 0-1.868-3.267A9 9 0 0 0 25 16a9 9 0 1 0-14.585 7.033A4.977 4.977 0 0 0 15 30h10a4.995 4.995 0 0 0 1.794-9.658M9 16a6.996 6.996 0 0 1 13.985-.297A6.9 6.9 0 0 0 20 15a7.04 7.04 0 0 0-6.794 5.342a5 5 0 0 0-1.644 1.048A6.97 6.97 0 0 1 9 16m16 12H15a2.995 2.995 0 0 1-.696-5.908l.658-.157l.099-.67a4.992 4.992 0 0 1 9.878 0l.099.67l.658.156A2.995 2.995 0 0 1 25 28"/></svg>`,
	"partly-cloudy-night": `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32"><path fill="currentColor" d="M30 19a4.97 4.97 0 0 0-3.206-4.658A6.971 6.971 0 0 0 13.758 12.9a13.14 13.14 0 0 1 .131-8.52A1.015 1.015 0 0 0 12.98 3a1 1 0 0 0-.175.016a13.096 13.096 0 0 0 1.825 25.981c.164.006.328 0 .49 0a13.04 13.04 0 0 0 10.29-5.038A4.99 4.99 0 0 0 30 19m-15.297 7.998a11.095 11.095 0 0 1-3.2-21.584a15.2 15.2 0 0 0 .844 9.367A4.988 4.988 0 0 0 15 24h7.677a11.1 11.1 0 0 1-7.556 2.998c-.138 0-.278.004-.418 0M25 22H15a2.995 2.995 0 0 1-.696-5.908l.658-.157l.099-.67a4.992 4.992 0 0 1 9.878 0l.099.67l.658.157A2.995 2.995 0 0 1 25 22"/></svg>`,
	"mostly-cloudy-day":   `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32"><path fill="currentColor" d="M21.743 18.692a6 6 0 0 0 1.057-1.086a5.998 5.998 0 1 0-10.733-4.445A7.56 7.56 0 0 0 6.35 18.25A5.993 5.993 0 0 0 8 30.005h11a5.985 5.985 0 0 0 2.743-11.313M18 10.005a4.004 4.004 0 0 1 4 4a3.96 3.96 0 0 1-.8 2.4a4 4 0 0 1-.94.891a7.54 7.54 0 0 0-6.134-4.24A4 4 0 0 1 18 10.006m1 18H8a3.993 3.993 0 0 1-.673-7.93l.663-.112l.146-.656a5.496 5.496 0 0 1 10.729 0l.146.656l.662.112a3.993 3.993 0 0 1-.673 7.93m7-15.001h4v2h-4zM22.95 7.64l2.828-2.827l1.415 1.414l-2.829 2.828zM17 2.005h2v4h-2zM8.808 6.227l1.414-1.414l2.829 2.828l-1.415 1.414z"/></svg>`,
	"mostly-cloudy-night": `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32"><path fill="currentColor" d="M29.844 15.035a1.52 1.52 0 0 0-1.23-.866a5.36 5.36 0 0 1-3.41-1.716a6.47 6.47 0 0 1-1.286-6.392a1.6 1.6 0 0 0-.299-1.546a1.45 1.45 0 0 0-1.36-.493l-.019.003a7.93 7.93 0 0 0-6.22 7.431A7.4 7.4 0 0 0 13.5 11a7.55 7.55 0 0 0-7.15 5.244A5.993 5.993 0 0 0 8 28h11a5.977 5.977 0 0 0 5.615-8.088a7.5 7.5 0 0 0 5.132-3.357a1.54 1.54 0 0 0 .097-1.52M19 26H8a3.993 3.993 0 0 1-.673-7.93l.663-.112l.145-.656a5.496 5.496 0 0 1 10.73 0l.145.656l.663.113A3.993 3.993 0 0 1 19 26m4.465-8.001h-.021a5.96 5.96 0 0 0-2.795-1.755a7.5 7.5 0 0 0-2.6-3.677c-.01-.101-.036-.197-.041-.3a6.08 6.08 0 0 1 3.79-6.05a8.46 8.46 0 0 0 1.94 7.596a7.4 7.4 0 0 0 3.902 2.228a5.43 5.43 0 0 1-4.175 1.958"/></svg>`,
	"light-rain":          `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32"><path fill="currentColor" d="M11 30a1 1 0 0 1-.894-1.447l2-4a1 1 0 1 1 1.788.894l-2 4A1 1 0 0 1 11 30"/><path fill="currentColor" d="M24.8 9.136a8.994 8.994 0 0 0-17.6 0A6.497 6.497 0 0 0 8.5 22h10.881l-1.276 2.553a1 1 0 0 0 1.789.894L21.618 22H23.5a6.497 6.497 0 0 0 1.3-12.864M23.5 20h-15a4.498 4.498 0 0 1-.356-8.981l.816-.064l.099-.812a6.994 6.994 0 0 1 13.883 0l.099.812l.815.064A4.498 4.498 0 0 1 23.5 20"/></svg>`,
	"rain":                `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32"><path fill="currentColor" d="M23.5 22h-15A6.5 6.5 0 0 1 7.2 9.14a9 9 0 0 1 17.6 0A6.5 6.5 0 0 1 23.5 22M16 4a7 7 0 0 0-6.94 6.14L9 11h-.86a4.5 4.5 0 0 0 .36 9h15a4.5 4.5 0 0 0 .36-9H23l-.1-.82A7 7 0 0 0 16 4m-2 26a.93.93 0 0 1-.45-.11a1 1 0 0 1-.44-1.34l2-4a1 1 0 1 1 1.78.9l-2 4A1 1 0 0 1 14 30m6 0a.93.93 0 0 1-.45-.11a1 1 0 0 1-.44-1.34l2-4a1 1 0 1 1 1.78.9l-2 4A1 1 0 0 1 20 30M8 30a.93.93 0 0 1-.45-.11a1 1 0 0 1-.44-1.34l2-4a1 1 0 1 1 1.78.9l-2 4A1 1 0 0 1 8 30"/></svg>`,
	"thunder":             `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32"><path fill="currentColor" d="M21 30a1 1 0 0 1-.894-1.447l2-4a1 1 0 1 1 1.788.894l-2 4A1 1 0 0 1 21 30M9 32a1 1 0 0 1-.894-1.447l2-4a1 1 0 1 1 1.788.894l-2 4A1 1 0 0 1 9 32m6.901-1.504l-1.736-.992L17.31 24h-6l4.855-8.496l1.736.992L14.756 22h6.001z"/><path fill="currentColor" d="M24.8 9.136a8.994 8.994 0 0 0-17.6 0a6.493 6.493 0 0 0 .23 12.768l-1.324 2.649a1 1 0 1 0 1.789.894l2-4a1 1 0 0 0-.447-1.341A1 1 0 0 0 9 20.01V20h-.5a4.498 4.498 0 0 1-.356-8.981l.816-.064l.099-.812a6.994 6.994 0 0 1 13.883 0l.099.812l.815.064A4.498 4.498 0 0 1 23.5 20H23v2h.5a6.497 6.497 0 0 0 1.3-12.864"/></svg>`,
	"snow":                `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32"><path fill="currentColor" d="M23.5 22h-15A6.5 6.5 0 0 1 7.2 9.14a9 9 0 0 1 17.6 0A6.5 6.5 0 0 1 23.5 22M16 4a7 7 0 0 0-6.94 6.14L9 11h-.86a4.5 4.5 0 0 0 .36 9h15a4.5 4.5 0 0 0 .36-9H23l-.1-.82A7 7 0 0 0 16 4m-4 21.05L10.95 24L9.5 25.45L8.05 24L7 25.05l1.45 1.45L7 27.95L8.05 29l1.45-1.45L10.95 29L12 27.95l-1.45-1.45zm14 0L24.95 24l-1.45 1.45L22.05 24L21 25.05l1.45 1.45L21 27.95L22.05 29l1.45-1.45L24.95 29L26 27.95l-1.45-1.45zm-7 2L17.95 26l-1.45 1.45L15.05 26L14 27.05l1.45 1.45L14 29.95L15.05 31l1.45-1.45L17.95 31L19 29.95l-1.45-1.45z"/></svg>`,
	"mist":                `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32"><path fill="currentColor" d="M24.8 11.138a8.994 8.994 0 0 0-17.6 0A6.53 6.53 0 0 0 2 17.5V19a1 1 0 0 0 1 1h12a1 1 0 0 0 0-2H4v-.497a4.52 4.52 0 0 1 4.144-4.482l.816-.064l.099-.812a6.994 6.994 0 0 1 13.883 0l.099.813l.815.063A4.496 4.496 0 0 1 23.5 22H7a1 1 0 0 0 0 2h16.5a6.496 6.496 0 0 0 1.3-12.862"/><rect width="18" height="2" x="2" y="26" fill="currentColor" rx="1"/></svg>`,
}

func getWeatherIcon(iconId string) string {
	switch iconId {
	case "01d":
		return WeatherIcons["clear-day"]
	case "01n":
		return WeatherIcons["clear-night"]
	case "02d", "03d":
		return WeatherIcons["partly-cloudy-day"]
	case "02n", "03n":
		return WeatherIcons["partly-cloudy-night"]
	case "04d":
		return WeatherIcons["mostly-cloudy-day"]
	case "04n":
		return WeatherIcons["mostly-cloudy-night"]
	case "09d", "09n":
		return WeatherIcons["light-rain"]
	case "10d", "10n":
		return WeatherIcons["rain"]
	case "11d", "11n":
		return WeatherIcons["thunder"]
	case "13d", "13n":
		return WeatherIcons["snow"]
	case "50d", "50n":
		return WeatherIcons["mist"]
	default:
		return ""
	}
}

func init() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found")
	}
}

func main() {
	if err := os.MkdirAll("public/uploads", 0755); err != nil {
		log.Fatal(err)
	}

	app, err := NewApp("passport.db?cache=shared&mode=rwc&_journal_mode=WAL")
	if err != nil {
		log.Fatal(err)
	}

	templatesDir, err := fs.Sub(embeddedAssets, "templates")
	if err != nil {
		log.Fatal(err)
	}

	assetsDir, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		log.Fatal(err)
	}

	css, err := fs.ReadFile(embeddedAssets, "assets/tailwind.css")

	if err != nil {
		log.Fatal(err)
	}

	engine := handlebars.NewFileSystem(http.FS(templatesDir), ".hbs")

	engine.AddFunc("inlineCSS", func() string {
		return string(css)
	})

	engine.AddFunc("devContent", func() string {
		return devContent
	})

	router := fiber.New(fiber.Config{
		Views: engine,
	})

	router.Use(helmet.New(helmet.ConfigDefault))

	router.Use("/", static.New("./public", static.Config{
		Browse: false,
		MaxAge: 31536000,
	}))

	router.Use("/assets", static.New("", static.Config{
		FS:     assetsDir,
		MaxAge: 31536000,
	}))

	router.Get("/", func(c fiber.Ctx) error {
		categories, err := app.GetCategories()
		if err != nil {
			return err
		}

		weather := app.WeatherCache.GetWeather()

		return c.Render("views/index", fiber.Map{
			"SearchProvider": os.Getenv("PASSPORT_SEARCH_PROVIDER"),
			"WeatherData": fiber.Map{
				"Temp": weather.Temperature,
				"Desc": weather.WeatherText,
				"Icon": getWeatherIcon(weather.Icon),
			},
			"Categories": categories,
		}, "layouts/main")
	})

	router.Use(middleware.AdminMiddleware(app.db))

	router.Get("/admin/login", func(c fiber.Ctx) error {
		if c.Locals("IsAdmin") != nil {
			return c.Redirect().To("/admin")
		}

		return c.Render("views/admin/login", fiber.Map{}, "layouts/main")
	})

	router.Post("/admin/login", func(c fiber.Ctx) error {
		if c.Locals("IsAdmin") != nil {
			return c.Redirect().To("/admin")
		}

		var loginData struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.Bind().JSON(&loginData); err != nil {
			return err
		}

		if loginData.Username != os.Getenv("PASSPORT_ADMIN_USERNAME") || loginData.Password != os.Getenv("PASSPORT_ADMIN_PASSWORD") {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid username or password"})
		}

		// Create new session
		sessionID := uuid.NewString()
		expiresAt := time.Now().Add(time.Hour * 24 * 7)
		_, err := app.db.Exec(`
			INSERT INTO sessions (session_id, expires_at) 
			VALUES (?, ?)
		`, sessionID, expiresAt)
		if err != nil {
			return err
		}

		// Set cookie
		c.Cookie(&fiber.Cookie{
			Name:    "SessionToken",
			Value:   sessionID,
			Expires: expiresAt,
		})

		return c.Status(http.StatusOK).JSON(fiber.Map{"message": "Logged in successfully"})
	})

	router.Get("/admin", func(c fiber.Ctx) error {
		if c.Locals("IsAdmin") == nil {
			return c.Redirect().To("/admin/login")
		}

		categories, err := app.GetCategories()
		if err != nil {
			return err
		}

		return c.Render("views/admin/index", fiber.Map{
			"Categories": categories,
		}, "layouts/main")
	})

	api := router.Group("/api")
	{
		api.Use(func(c fiber.Ctx) error {
			if c.Locals("IsAdmin") == nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
			}
			return c.Next()
		})

		api.Post("/categories", CreateCategory(app.db))
		api.Post("/links", CreateLink(app.db))

		api.Delete("/links/:id", func(c fiber.Ctx) error {
			id := c.Params("id")

			var icon string
			if err := app.db.QueryRow("SELECT icon FROM links WHERE id = ?", id).Scan(&icon); err != nil {
				return err
			}

			_, err := app.db.Exec("DELETE FROM links WHERE id = ?", id)
			if err != nil {
				return err
			}

			if icon != "" {
				if err := os.Remove(filepath.Join("public/", icon)); err != nil {
					return err
				}
			}
			return c.SendStatus(fiber.StatusOK)
		})

		api.Delete("/categories/:id", func(c fiber.Ctx) error {
			id := c.Params("id")

			rows, err := app.db.Query(`
				SELECT icon FROM categories WHERE id = ?
				UNION
				SELECT icon FROM links WHERE category_id = ?
			`, id, id)

			if err != nil {
				return err
			}

			defer rows.Close()

			var icons []string
			for rows.Next() {
				var icon string
				if err := rows.Scan(&icon); err != nil {
					return err
				}
				icons = append(icons, icon)
			}

			tx, err := app.db.Begin()
			if err != nil {
				return err
			}
			defer tx.Rollback()

			_, err = tx.Exec("DELETE FROM categories WHERE id = ?", id)
			if err != nil {
				return err
			}

			_, err = tx.Exec("DELETE FROM links WHERE category_id = ?", id)
			if err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return err
			}

			for _, icon := range icons {
				if icon == "" {
					continue
				}

				if err := os.Remove(filepath.Join("public/", icon)); err != nil {
					return err
				}
			}

			return c.SendStatus(fiber.StatusOK)
		})
	}

	router.Listen(":3000")
}
