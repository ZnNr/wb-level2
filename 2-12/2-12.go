package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

/*
Утилита grep
Реализовать утилиту фильтрации текстового потока (аналог команды grep).

Программа должна читать входной поток (STDIN или файл) и выводить строки, соответствующие заданному шаблону (подстроке или регулярному выражению).

Необходимо поддерживать следующие флаги:

-A N — после каждой найденной строки дополнительно вывести N строк после неё (контекст).

-B N — вывести N строк до каждой найденной строки.

-C N — вывести N строк контекста вокруг найденной строки (включает и до, и после; эквивалентно -A N -B N).

-c — выводить только то количество строк, что совпадающих с шаблоном (т.е. вместо самих строк — число).

-i — игнорировать регистр.

-v — инвертировать фильтр: выводить строки, не содержащие шаблон.

-F — воспринимать шаблон как фиксированную строку, а не регулярное выражение (т.е. выполнять точное совпадение подстроки).

-n — выводить номер строки перед каждой найденной строкой.

Программа должна поддерживать сочетания флагов (например, -C 2 -n -i – 2 строки контекста, вывод номеров, без учета регистра и т.д.).

Результат работы должен максимально соответствовать поведению команды UNIX grep.

Обязательно учесть пограничные случаи (начало/конец файла для контекста, повторяющиеся совпадения и пр.).

Код должен быть чистым, отформатированным (gofmt), работать без ситуаций гонки и успешно проходить golint.
*/

// grepOptions хранит все параметры командной строки
type grepOptions struct {
	afterContext  int
	beforeContext int
	countOnly     bool
	ignoreCase    bool
	invertMatch   bool
	fixedString   bool
	lineNumber    bool
	pattern       string
}

func main() {
	var opts grepOptions

	flag.IntVar(&opts.afterContext, "A", 0, "Print N lines of trailing context after matching lines")
	flag.IntVar(&opts.beforeContext, "B", 0, "Print N lines of leading context before matching lines")
	flag.IntVar(&opts.afterContext, "C", 0, "Print N lines of output context")
	flag.IntVar(&opts.beforeContext, "C", 0, "Print N lines of input context")
	flag.BoolVar(&opts.countOnly, "c", false, "Print only a count of matching lines")
	flag.BoolVar(&opts.ignoreCase, "i", false, "Ignore case distinctions")
	flag.BoolVar(&opts.invertMatch, "v", false, "Invert the sense of matching")
	flag.BoolVar(&opts.fixedString, "F", false, "Interpret pattern as a fixed string")
	flag.BoolVar(&opts.lineNumber, "n", false, "Print line number with output lines")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [OPTION]... PATTERN [FILE]...\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "error: missing pattern\n")
		os.Exit(1)
	}

	opts.pattern = args[0]
	var files []string
	if len(args) > 1 {
		files = args[1:]
	} else {
		files = []string{"-"} // STDIN
	}

	if opts.afterContext != 0 && opts.beforeContext == 0 {
		opts.beforeContext = 0
	}
	if opts.beforeContext != 0 && opts.afterContext == 0 {
		opts.afterContext = 0
	}
	// Если задан -C N, он устанавливает оба контекста через механизм выше через дублирование флага

	exitCode := 0
	for _, filename := range files {
		var r io.Reader
		if filename == "-" {
			r = os.Stdin
		} else {
			f, err := os.Open(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				exitCode = 1
				continue
			}
			defer f.Close()
			r = f
		}

		found, err := grep(r, opts, filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			exitCode = 1
		}
		if found {
			exitCode = 0 // хотя бы в одном файле совпадения есть
		}
	}

	os.Exit(exitCode)
}

// matcher определяет, совпадает ли строка с шаблоном
type matcher func(line string) bool

func buildMatcher(pattern string, opts grepOptions) (matcher, error) {
	if opts.fixedString {
		searchStr := pattern
		if opts.ignoreCase {
			searchStr = strings.ToLower(searchStr)
			return func(line string) bool {
				return strings.Contains(strings.ToLower(line), searchStr)
			}, nil
		}
		return func(line string) bool {
			return strings.Contains(line, searchStr)
		}, nil
	}

	// regex
	reFlags := ""
	if opts.ignoreCase {
		reFlags = "(?i)"
	}
	re, err := regexp.Compile(reFlags + pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}
	return func(line string) bool {
		return re.MatchString(line)
	}, nil
}

// grep выполняет поиск по одному входному потоку
func grep(r io.Reader, opts grepOptions, filename string) (bool /*found any*/, error) {
	scanner := bufio.NewScanner(r)
	matchFn, err := buildMatcher(opts.pattern, opts)
	if err != nil {
		return false, err
	}

	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}

	totalLines := len(lines)
	matchingLines := make([]bool, totalLines)
	matchCount := 0

	for i, line := range lines {
		matches := matchFn(line)
		if opts.invertMatch {
			matches = !matches
		}
		matchingLines[i] = matches
		if matches {
			matchCount++
		}
	}

	if opts.countOnly {
		fmt.Println(matchCount)
		return matchCount > 0, nil
	}

	if matchCount == 0 {
		return false, nil
	}

	// Собираем диапазоны строк для вывода с учётом контекста
	outputLines := make(map[int]bool)
	for i, isMatch := range matchingLines {
		if isMatch {
			// Добавляем контекст до
			start := i - opts.beforeContext
			if start < 0 {
				start = 0
			}
			// Добавляем контекст после
			end := i + opts.afterContext
			if end >= totalLines {
				end = totalLines - 1
			}
			for j := start; j <= end; j++ {
				outputLines[j] = true
			}
		}
	}

	// Преобразуем map в отсортированный срез индексов
	var indices []int
	for idx := range outputLines {
		indices = append(indices, idx)
	}
	sortInts(indices)

	// Выводим
	prevEnd := -2 // чтобы отслеживать разрывы
	for _, idx := range indices {
		line := lines[idx]
		isMatch := matchingLines[idx]

		// Выводим разделитель, если есть разрыв в контексте
		if opts.beforeContext > 0 || opts.afterContext > 0 {
			if prevEnd != idx-1 && prevEnd != -2 {
				fmt.Println("--")
			}
			prevEnd = idx
		}

		// Формируем префикс
		prefix := ""
		if opts.lineNumber {
			prefix = strconv.Itoa(idx+1) + ":"
			if !isMatch {
				prefix = strconv.Itoa(idx+1) + "-"
			}
		}

		// Для многофайлового режима добавляем имя файла
		filePrefix := ""
		if filename != "-" && (len(flag.Args()) > 2 || (len(flag.Args()) == 2 && flag.Args()[1] != "-")) {
			filePrefix = filename + ":"
			if !isMatch {
				filePrefix = filename + "-"
			}
		}

		fmt.Print(filePrefix + prefix + line + "\n")
	}

	return true, nil
}

// sortInts сортирует срез целых чисел по возрастанию
func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}
