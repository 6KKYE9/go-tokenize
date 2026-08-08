package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	lower := flag.Bool("lower", true, "统一转小写")
	minLen := flag.Int("min", 1, "少于这么多字符的词丢掉")
	stop := flag.Bool("stop", false, "过滤常见停用词")
	stopFile := flag.String("stopfile", "", "从文件读停用词，一行一个")
	gram := flag.Int("cjk", 1, "中日韩文字按几个字一组切")
	ngram := flag.Int("ngram", 0, "统计 n 个相邻词的组合，0 表示不统计")
	top := flag.Int("top", 20, "只显示前几个，0 表示全部")
	sum := flag.Bool("sum", false, "只输出统计概览")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `go-tokenize - 中英文分词和词频统计

  go-tokenize -top 10 article.txt
  type article.txt | go-tokenize -stop -min 2
  go-tokenize -cjk 2 -top 15 中文.txt     中文按两字一组切
  go-tokenize -ngram 2 -top 10 en.txt     看常见的两词搭配

参数:
`)
		flag.PrintDefaults()
	}
	flag.Parse()

	text, err := readInput(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, "读输入出错:", err)
		os.Exit(1)
	}
	if strings.TrimSpace(text) == "" {
		fmt.Fprintln(os.Stderr, "没读到内容")
		os.Exit(1)
	}

	opt := Options{Lower: *lower, MinLen: *minLen, CJKGram: *gram}
	if *stop {
		opt.Stopwords = DefaultStopwords
	}
	if *stopFile != "" {
		ws, err := readStopwords(*stopFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "读停用词出错:", err)
			os.Exit(1)
		}
		opt.Stopwords = append(opt.Stopwords, ws...)
	}

	tk := NewTokenizer(opt)

	if *sum {
		s := tk.Summarize(text)
		fmt.Printf("字符 %d（其中中日韩 %d）  字节 %d  行 %d\n", s.Chars, s.CJKChars, s.Bytes, s.Lines)
		fmt.Printf("词数 %d  不重复 %d", s.Words, s.Unique)
		if s.Words > 0 {
			fmt.Printf("  重复率 %.1f%%", (1-float64(s.Unique)/float64(s.Words))*100)
		}
		fmt.Println()
		return
	}

	var counts []Count
	if *ngram > 1 {
		counts = tk.Ngrams(text, *ngram)
	} else {
		counts = tk.Count(text)
	}

	if len(counts) == 0 {
		fmt.Fprintln(os.Stderr, "一个词都没切出来")
		os.Exit(1)
	}

	shown := counts
	if *top > 0 && *top < len(shown) {
		shown = shown[:*top]
	}

	total := 0
	for _, c := range counts {
		total += c.N
	}

	maxW, maxN := 0, shown[0].N
	for _, c := range shown {
		if n := len([]rune(c.Word)); n > maxW {
			maxW = n
		}
	}
	if maxW > 30 {
		maxW = 30
	}

	for _, c := range shown {
		w := c.Word
		if len([]rune(w)) > maxW {
			w = string([]rune(w)[:maxW])
		}
		bar := c.N * 30 / maxN
		if bar == 0 {
			bar = 1
		}
		// 中文字符宽度是两倍，用 rune 数补齐才对得整齐
		pad := maxW - len([]rune(w))
		fmt.Printf("%s%s │%-30s %d (%.2f%%)\n",
			w, strings.Repeat(" ", pad), strings.Repeat("█", bar),
			c.N, float64(c.N)/float64(total)*100)
	}
	fmt.Fprintf(os.Stderr, "\n共 %d 个词，%d 种\n", total, len(counts))
}

func readInput(args []string) (string, error) {
	if len(args) == 0 {
		b, err := io.ReadAll(os.Stdin)
		return string(b), err
	}
	var sb strings.Builder
	for _, p := range args {
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		sb.Write(b)
		sb.WriteByte('\n')
	}
	return sb.String(), nil
}

func readStopwords(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	return out, nil
}
