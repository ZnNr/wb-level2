package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

/*
Утилита sort
Реализовать упрощённый аналог UNIX-утилиты sort (сортировка строк).

Программа должна читать строки (из файла или STDIN) и выводить их отсортированными.

Обязательные флаги (как в GNU sort):

-k N — сортировать по столбцу (колонке) №N (разделитель — табуляция по умолчанию).
Например, «sort -k 2» отсортирует строки по второму столбцу каждой строки.

-n — сортировать по числовому значению (строки интерпретируются как числа).

-r — сортировать в обратном порядке (reverse).

-u — не выводить повторяющиеся строки (только уникальные).

Дополнительные флаги:

-M — сортировать по названию месяца (Jan, Feb, ... Dec), т.е. распознавать специфический формат дат.

-b — игнорировать хвостовые пробелы (trailing blanks).

-c — проверить, отсортированы ли данные; если нет, вывести сообщение об этом.

-h — сортировать по числовому значению с учётом суффиксов (например, К = килобайт, М = мегабайт — человекочитаемые размеры).

Программа должна корректно обрабатывать комбинации флагов (например, -nr — числовая сортировка в обратном порядке, и т.д.).

Необходимо предусмотреть эффективную обработку больших файлов.

Код должен проходить все тесты, а также проверки go vet и golint (понимание, что требуются надлежащие комментарии, имена и структура программы).
*/

// monthMap maps month abbreviations to their numeric order.
var monthMap = map[string]int{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
	"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
}

// parseHumanReadable parses human-readable sizes like "1K", "2M", "512", "3.2G".
func parseHumanReadable(s string) (float64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	// Try to parse as float first (no suffix)
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return val, nil
	}

	// Find where number ends and suffix begins
	var i int
	for i = len(s) - 1; i >= 0 && !unicode.IsDigit(rune(s[i])); i-- {
	}
	if i < 0 {
		return 0, fmt.Errorf("no digits in %q", s)
	}

	numPart := s[:i+1]
	suffix := s[i+1:]

	num, err := strconv.ParseFloat(numPart, 64)
	if err != nil {
		return 0, err
	}

	switch suffix {
	case "", "b":
		return num, nil
	case "k":
		return num * 1024, nil
	case "m":
		return num * 1024 * 1024, nil
	case "g":
		return num * 1024 * 1024 * 1024, nil
	case "t":
		return num * 1024 * 1024 * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("unknown suffix in %q", s)
	}
}

// getFieldValue extracts the N-th field (1-based) from a line, split by tab.
// If line has fewer fields, returns the whole line.
func getFieldValue(line string, n int) string {
	if n <= 0 {
		return line
	}
	fields := strings.Split(line, "\t")
	if n > len(fields) {
		return line
	}
	return fields[n-1]
}

// trimTrailingBlanks removes trailing spaces and tabs.
func trimTrailingBlanks(s string) string {
	return strings.TrimRightFunc(s, func(r rune) bool {
		return r == ' ' || r == '\t'
	})
}

// monthValue returns numeric month value or 0 if not a valid month.
func monthValue(s string) int {
	s = strings.ToLower(strings.TrimSpace(s))
	if v, ok := monthMap[s]; ok {
		return v
	}
	return 0
}

func main() {
	var (
		keyNum       = flag.Int("k", 0, "sort by column N (1-based, tab-separated)")
		numeric      = flag.Bool("n", false, "compare according to string numerical value")
		reverse      = flag.Bool("r", false, "reverse the result of comparisons")
		unique       = flag.Bool("u", false, "output only the first of an equal run")
		monthSort    = flag.Bool("M", false, "compare according to month name")
		ignoreBlanks = flag.Bool("b", false, "ignore trailing blanks")
		check        = flag.Bool("c", false, "check whether input is sorted; exit with status 1 if not")
		human        = flag.Bool("h", false, "compare human readable numbers (e.g., 2K, 1G)")
	)

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [OPTION]... [FILE]...\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Write sorted lines to standard output.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Validate flags
	if *keyNum < 0 {
		fmt.Fprintf(os.Stderr, "Invalid column number: %d\n", *keyNum)
		os.Exit(1)
	}

	// Determine input source
	var reader io.Reader
	if flag.NArg() == 0 {
		reader = os.Stdin
	} else {
		file, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		reader = file
	}

	// Read all lines
	scanner := bufio.NewScanner(reader)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	// Apply -b: trim trailing blanks if needed
	if *ignoreBlanks {
		for i, line := range lines {
			lines[i] = trimTrailingBlanks(line)
		}
	}

	// Define comparison function
	lessFunc := func(i, j int) bool {
		a, b := lines[i], lines[j]

		// Extract field if -k is used
		if *keyNum > 0 {
			a = getFieldValue(a, *keyNum)
			b = getFieldValue(b, *keyNum)
		}

		var less bool
		switch {
		case *monthSort:
			ma, mb := monthValue(a), monthValue(b)
			if ma != 0 || mb != 0 {
				less = ma < mb
			} else {
				less = a < b // fallback to lexicographic
			}
		case *human:
			va, errA := parseHumanReadable(a)
			vb, errB := parseHumanReadable(b)
			if errA == nil && errB == nil {
				less = va < vb
			} else {
				less = a < b // fallback
			}
		case *numeric:
			va, errA := strconv.ParseFloat(a, 64)
			vb, errB := strconv.ParseFloat(b, 64)
			if errA == nil && errB == nil {
				less = va < vb
			} else {
				less = a < b // fallback
			}
		default:
			less = a < b
		}

		if *reverse {
			return !less
		}
		return less
	}

	if *check {
		// Check if already sorted
		for i := 1; i < len(lines); i++ {
			if lessFunc(i, i-1) { // if lines[i] < lines[i-1] → not sorted
				fmt.Fprintf(os.Stderr, "Input is not sorted.\n")
				os.Exit(1)
			}
		}
		// If we reach here, it's sorted → exit successfully
		os.Exit(0)
	}

	// Sort
	sort.SliceStable(lines, lessFunc)

	// Apply -u: deduplicate consecutive equal lines (after sorting)
	if *unique {
		if len(lines) == 0 {
			return
		}
		uniqueLines := []string{lines[0]}
		for i := 1; i < len(lines); i++ {
			// Compare full lines (not just key field) for uniqueness
			if lines[i] != lines[i-1] {
				uniqueLines = append(uniqueLines, lines[i])
			}
		}
		lines = uniqueLines
	}

	// Output
	for _, line := range lines {
		fmt.Println(line)
	}
}
