/*
Утилита cut
Реализовать утилиту, которая считывает входные данные (STDIN) и разбивает каждую строку по заданному разделителю, после чего выводит определённые поля (колонки).

Аналог команды cut с поддержкой флагов:

-f "fields" — указание номеров полей (колонок), которые нужно вывести. Номера через запятую, можно диапазоны.
Например: «-f 1,3-5» — вывести 1-й и с 3-го по 5-й столбцы.

-d "delimiter" — использовать другой разделитель (символ). По умолчанию разделитель — табуляция ('\t').

-s – (separated) только строки, содержащие разделитель. Если флаг указан, то строки без разделителя игнорируются (не выводятся).

Программа должна корректно парсить аргументы, поддерживать различные комбинации (например, несколько отдельных полей и диапазонов), учитывать, что номера полей могут выходить за границы (в таком случае эти поля просто игнорируются).

Стоит обратить внимание на эффективность при обработке больших файлов. Все стандартные требования по качеству кода и тестам также применимы.
*/

package main

import (
	_ "bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config содержит конфигурацию программы
type Config struct {
	URL           string
	MaxDepth      int
	MaxWorkers    int
	OutputDir     string
	SameDomain    bool
	Timeout       time.Duration
	UserAgent     string
	VisitedURLs   *sync.Map
	DownloadQueue chan DownloadTask
	WaitGroup     sync.WaitGroup
	Client        *http.Client
}

// DownloadTask представляет задачу на скачивание
type DownloadTask struct {
	URL   string
	Depth int
	Type  string // html, css, js, image, other
}

// HTMLParser упрощенный парсер HTML
type HTMLParser struct {
	Content []byte
	Pos     int
}

// NewHTMLParser создает новый парсер
func NewHTMLParser(content []byte) *HTMLParser {
	return &HTMLParser{Content: content, Pos: 0}
}

// FindAll находит все значения атрибутов по тегу и атрибуту
func (p *HTMLParser) FindAll(tag, attribute string) []string {
	var results []string
	content := string(p.Content)
	tagStart := "<" + tag
	tagLen := len(tagStart)

	for i := 0; i < len(content); i++ {
		if i+tagLen <= len(content) && strings.EqualFold(content[i:i+tagLen], tagStart) {
			// Нашли начало тега
			j := i + tagLen
			for j < len(content) && content[j] != '>' {
				j++
			}
			if j < len(content) {
				tagContent := content[i : j+1]

				// Ищем атрибут
				attrStart := attribute + "=\""
				attrIdx := strings.Index(strings.ToLower(tagContent), attrStart)
				if attrIdx == -1 {
					// Попробуем с одинарными кавычками
					attrStart = attribute + "='"
					attrIdx = strings.Index(strings.ToLower(tagContent), attrStart)
					if attrIdx == -1 {
						i = j
						continue
					}
				}

				start := attrIdx + len(attrStart)
				end := start
				for end < len(tagContent) && tagContent[end] != '"' && tagContent[end] != '\'' {
					end++
				}

				if end < len(tagContent) {
					href := tagContent[start:end]
					if href != "" && !strings.HasPrefix(href, "#") &&
						!strings.HasPrefix(strings.ToLower(href), "javascript:") {
						results = append(results, href)
					}
				}
			}
			i = j
		}
	}
	return results
}

func main() {
	var (
		urlStr     = flag.String("url", "", "URL для скачивания")
		maxDepth   = flag.Int("depth", 1, "Максимальная глубина рекурсии")
		maxWorkers = flag.Int("workers", 5, "Количество параллельных загрузчиков")
		outputDir  = flag.String("output", "./mirror", "Директория для сохранения")
		timeout    = flag.Int("timeout", 30, "Таймаут запросов в секундах")
	)
	flag.Parse()

	if *urlStr == "" {
		fmt.Println("Ошибка: необходимо указать URL")
		fmt.Println("Использование: wget -url <URL> [-depth <глубина>] [-workers <количество>]")
		os.Exit(1)
	}

	// Парсинг URL для проверки
	_, err := url.Parse(*urlStr)
	if err != nil {
		fmt.Printf("Ошибка парсинга URL: %v\n", err)
		os.Exit(1)
	}

	config := &Config{
		URL:         *urlStr,
		MaxDepth:    *maxDepth,
		MaxWorkers:  *maxWorkers,
		OutputDir:   *outputDir,
		SameDomain:  true,
		Timeout:     time.Duration(*timeout) * time.Second,
		UserAgent:   "Go-Wget/1.0",
		VisitedURLs: &sync.Map{},
		Client: &http.Client{
			Timeout: time.Duration(*timeout) * time.Second,
		},
		DownloadQueue: make(chan DownloadTask, 1000),
	}

	// Создаем выходную директорию
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		fmt.Printf("Ошибка создания директории: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Начинаем скачивание %s (глубина: %d)\n", config.URL, config.MaxDepth)
	fmt.Printf("Сохранение в: %s\n", config.OutputDir)

	// Запускаем воркеры
	for i := 0; i < config.MaxWorkers; i++ {
		config.WaitGroup.Add(1)
		go worker(config, i)
	}

	// Добавляем начальную задачу
	config.DownloadQueue <- DownloadTask{
		URL:   config.URL,
		Depth: 0,
		Type:  "html",
	}

	// Ожидаем завершения всех задач
	config.WaitGroup.Wait()
	close(config.DownloadQueue)

	fmt.Println("Скачивание завершено!")
}

// worker обрабатывает задачи скачивания
func worker(config *Config, id int) {
	defer config.WaitGroup.Done()

	for task := range config.DownloadQueue {
		fmt.Printf("[Воркер %d] Скачивание: %s (глубина: %d)\n", id, task.URL, task.Depth)

		// Проверяем, не скачивали ли мы уже этот URL
		if _, visited := config.VisitedURLs.Load(task.URL); visited {
			continue
		}
		config.VisitedURLs.Store(task.URL, true)

		// Скачиваем ресурс
		content, contentType, err := downloadResource(config, task.URL)
		if err != nil {
			fmt.Printf("[Воркер %d] Ошибка скачивания %s: %v\n", id, task.URL, err)
			continue
		}

		// Сохраняем файл
		localPath, err := saveResource(config, task.URL, content, contentType)
		if err != nil {
			fmt.Printf("[Воркер %d] Ошибка сохранения %s: %v\n", id, task.URL, err)
			continue
		}

		// Если это HTML и не достигнута максимальная глубина - парсим ссылки
		if isHTMLContent(contentType) && task.Depth < config.MaxDepth {
			baseURL, _ := url.Parse(task.URL)
			links, resources := parseHTMLSimple(content, baseURL)

			// Добавляем новые задачи в очередь
			for _, link := range links {
				if shouldDownload(config, link, baseURL) {
					if _, visited := config.VisitedURLs.Load(link); !visited {
						config.DownloadQueue <- DownloadTask{
							URL:   link,
							Depth: task.Depth + 1,
							Type:  "html",
						}
					}
				}
			}

			// Добавляем ресурсы (CSS, JS, изображения)
			for _, res := range resources {
				if _, visited := config.VisitedURLs.Load(res); !visited {
					config.DownloadQueue <- DownloadTask{
						URL:   res,
						Depth: task.Depth + 1,
						Type:  "resource",
					}
				}
			}

			// Обновляем HTML с локальными путями
			updatedHTML := replaceLinksSimple(content, baseURL, localPath)
			if err := os.WriteFile(localPath, updatedHTML, 0644); err != nil {
				fmt.Printf("[Воркер %d] Ошибка обновления HTML: %v\n", id, err)
			}
		}
	}
}

// downloadResource скачивает ресурс по URL
func downloadResource(config *Config, urlStr string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", config.UserAgent)

	resp, err := config.Client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP статус: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return content, resp.Header.Get("Content-Type"), nil
}

// saveResource сохраняет ресурс в файл
func saveResource(config *Config, urlStr string, content []byte, contentType string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// Создаем путь для сохранения
	path := parsedURL.Path
	if path == "" || strings.HasSuffix(path, "/") {
		path = path + "index.html"
	}

	// Убираем начальный слэш
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	// Создаем директории
	fullPath := filepath.Join(config.OutputDir, parsedURL.Hostname(), path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Сохраняем файл
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return "", err
	}

	return fullPath, nil
}

// parseHTMLSimple парсит HTML и извлекает ссылки и ресурсы
func parseHTMLSimple(content []byte, baseURL *url.URL) ([]string, []string) {
	var links, resources []string
	contentStr := string(content)

	// Ищем ссылки в <a> тегах
	links = append(links, extractLinks(contentStr, "a", "href", baseURL)...)

	// Ищем ресурсы
	resources = append(resources, extractLinks(contentStr, "link", "href", baseURL)...)
	resources = append(resources, extractLinks(contentStr, "script", "src", baseURL)...)
	resources = append(resources, extractLinks(contentStr, "img", "src", baseURL)...)
	resources = append(resources, extractLinks(contentStr, "iframe", "src", baseURL)...)
	resources = append(resources, extractLinks(contentStr, "embed", "src", baseURL)...)
	resources = append(resources, extractLinks(contentStr, "source", "src", baseURL)...)

	// Убираем дубликаты
	links = removeDuplicates(links)
	resources = removeDuplicates(resources)

	return links, resources
}

// extractLinks извлекает ссылки из HTML
func extractLinks(content, tag, attr string, baseURL *url.URL) []string {
	var results []string
	tagStart := "<" + strings.ToLower(tag)
	attrPattern := strings.ToLower(attr) + "=\""

	pos := 0
	for {
		// Ищем начало тега
		tagPos := strings.Index(strings.ToLower(content[pos:]), tagStart)
		if tagPos == -1 {
			break
		}
		tagPos += pos

		// Ищем конец тега
		endPos := strings.Index(content[tagPos:], ">")
		if endPos == -1 {
			break
		}
		endPos += tagPos

		// Извлекаем атрибут
		tagContent := content[tagPos:endPos]
		attrPos := strings.Index(strings.ToLower(tagContent), attrPattern)
		if attrPos != -1 {
			start := tagPos + attrPos + len(attrPattern)
			end := start
			for end < len(content) && content[end] != '"' {
				end++
			}
			if end < len(content) {
				href := content[start:end]
				if isValidLink(href) {
					absoluteURL := resolveURL(href, baseURL)
					if absoluteURL != "" {
						results = append(results, absoluteURL)
					}
				}
			}
		}

		pos = endPos + 1
		if pos >= len(content) {
			break
		}
	}

	return results
}

// isValidLink проверяет, является ли ссылка валидной для скачивания
func isValidLink(link string) bool {
	return link != "" &&
		!strings.HasPrefix(link, "#") &&
		!strings.HasPrefix(strings.ToLower(link), "javascript:") &&
		!strings.HasPrefix(strings.ToLower(link), "mailto:")
}

// resolveURL преобразует относительный URL в абсолютный
func resolveURL(href string, baseURL *url.URL) string {
	if href == "" {
		return ""
	}

	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}

	// Абсолютные URL
	if parsed.IsAbs() {
		return parsed.String()
	}

	// Относительные URL
	resolved := baseURL.ResolveReference(parsed)
	return resolved.String()
}

// removeDuplicates удаляет дубликаты из среза
func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// shouldDownload проверяет, нужно ли скачивать ссылку
func shouldDownload(config *Config, urlStr string, baseURL *url.URL) bool {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// Проверяем схему
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	// Если включен режим только того же домена
	if config.SameDomain && parsed.Hostname() != baseURL.Hostname() {
		return false
	}

	return true
}

// isHTMLContent проверяет, является ли контент HTML
func isHTMLContent(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/html") ||
		strings.Contains(strings.ToLower(contentType), "application/xhtml+xml")
}

// replaceLinksSimple заменяет ссылки в HTML на локальные пути
func replaceLinksSimple(content []byte, baseURL *url.URL, localPath string) []byte {
	contentStr := string(content)

	// Заменяем ссылки в разных тегах
	tags := []struct {
		tag  string
		attr string
	}{
		{"a", "href"},
		{"link", "href"},
		{"script", "src"},
		{"img", "src"},
		{"iframe", "src"},
		{"embed", "src"},
		{"source", "src"},
	}

	for _, t := range tags {
		contentStr = replaceLinksInTag(contentStr, t.tag, t.attr, baseURL)
	}

	return []byte(contentStr)
}

// replaceLinksInTag заменяет ссылки в конкретном теге
func replaceLinksInTag(content, tag, attr string, baseURL *url.URL) string {
	tagStart := "<" + strings.ToLower(tag)
	attrPattern := strings.ToLower(attr) + "=\""

	var result strings.Builder
	pos := 0

	for {
		// Ищем начало тега
		tagPos := strings.Index(strings.ToLower(content[pos:]), tagStart)
		if tagPos == -1 {
			result.WriteString(content[pos:])
			break
		}
		tagPos += pos

		// Пишем все до тега
		result.WriteString(content[pos:tagPos])

		// Ищем конец тега
		endPos := strings.Index(content[tagPos:], ">")
		if endPos == -1 {
			result.WriteString(content[tagPos:])
			break
		}
		endPos += tagPos

		// Извлекаем и заменяем атрибут
		tagContent := content[tagPos:endPos]
		attrPos := strings.Index(strings.ToLower(tagContent), attrPattern)
		if attrPos != -1 {
			start := attrPos + len(attrPattern)
			end := start
			for end < len(tagContent) && tagContent[end] != '"' {
				end++
			}
			if end < len(tagContent) {
				href := tagContent[start:end]
				if isValidLink(href) {
					absoluteURL := resolveURL(href, baseURL)
					if absoluteURL != "" {
						// Преобразуем в локальный путь
						localPath := urlToLocalPath(absoluteURL, baseURL)
						// Заменяем ссылку
						newTag := tagContent[:start] + localPath + tagContent[end:]
						result.WriteString(newTag)
					} else {
						result.WriteString(tagContent)
					}
				} else {
					result.WriteString(tagContent)
				}
			} else {
				result.WriteString(tagContent)
			}
		} else {
			result.WriteString(tagContent)
		}

		pos = endPos
		if pos >= len(content) {
			break
		}
	}

	return result.String()
}

// urlToLocalPath преобразует URL в локальный путь
func urlToLocalPath(urlStr string, baseURL *url.URL) string {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}

	// Если другой домен - оставляем как есть
	if parsed.Hostname() != baseURL.Hostname() {
		return urlStr
	}

	// Преобразуем путь
	path := parsed.Path
	if path == "" || strings.HasSuffix(path, "/") {
		path = path + "index.html"
	}

	// Убираем начальный слэш
	if strings.HasPrefix(path, "/") {
		path = "." + path
	}

	// Добавляем query если есть
	if parsed.RawQuery != "" {
		path = path + "?" + parsed.RawQuery
	}

	// Добавляем fragment если есть
	if parsed.Fragment != "" {
		path = path + "#" + parsed.Fragment
	}

	return path
}

// hashContent создает хеш содержимого
func hashContent(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash[:16])
}
