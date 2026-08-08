package main

import (
	"sort"
	"strings"
	"unicode"
)

type Options struct {
	Lower     bool     // 统一转小写
	MinLen    int      // 少于这个长度的词丢掉
	Stopwords []string // 停用词
	CJKGram   int      // 中日韩文字按几个字一组切，0 表示按单字
}

type Tokenizer struct {
	opt  Options
	stop map[string]bool
}

func NewTokenizer(opt Options) *Tokenizer {
	t := &Tokenizer{opt: opt, stop: map[string]bool{}}
	for _, w := range opt.Stopwords {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		// 停用词也要按同样的规则规范化，不然大小写对不上
		if opt.Lower {
			w = strings.ToLower(w)
		}
		t.stop[w] = true
	}
	if t.opt.CJKGram < 1 {
		t.opt.CJKGram = 1
	}
	return t
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r)
}

// 词的一部分：字母、数字，还有词内的连字符和撇号
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// 切词。英文按空白和标点切，中日韩按 n-gram 切
func (t *Tokenizer) Split(s string) []string {
	var out []string
	rs := []rune(s)
	i := 0

	for i < len(rs) {
		r := rs[i]

		if isCJK(r) {
			// 收集连续的一段 CJK
			j := i
			for j < len(rs) && isCJK(rs[j]) {
				j++
			}
			out = append(out, t.gram(rs[i:j])...)
			i = j
			continue
		}

		if isWordRune(r) {
			j := i
			for j < len(rs) {
				if isWordRune(rs[j]) {
					j++
					continue
				}
				// don't、well-known 这种词内的符号要保留，但结尾的不要
				if (rs[j] == '\'' || rs[j] == '-' || rs[j] == '’') &&
					j+1 < len(rs) && isWordRune(rs[j+1]) {
					j += 2
					continue
				}
				break
			}
			out = append(out, string(rs[i:j]))
			i = j
			continue
		}

		i++
	}
	return out
}

// CJK 按 n 个字一组切，滑动窗口
func (t *Tokenizer) gram(rs []rune) []string {
	n := t.opt.CJKGram
	if n <= 1 || len(rs) <= n {
		if n > 1 && len(rs) >= 1 {
			// 整段还没一个 gram 长，整段当一个词
			return []string{string(rs)}
		}
		out := make([]string, len(rs))
		for i, r := range rs {
			out[i] = string(r)
		}
		return out
	}
	out := make([]string, 0, len(rs)-n+1)
	for i := 0; i+n <= len(rs); i++ {
		out = append(out, string(rs[i:i+n]))
	}
	return out
}

// 切词 + 过滤
func (t *Tokenizer) Tokens(s string) []string {
	raw := t.Split(s)
	out := make([]string, 0, len(raw))
	for _, w := range raw {
		if t.opt.Lower {
			w = strings.ToLower(w)
		}
		// MinLen 按字符数算，不是字节数，不然中文会被全滤掉
		if t.opt.MinLen > 0 && len([]rune(w)) < t.opt.MinLen {
			continue
		}
		if t.stop[w] {
			continue
		}
		out = append(out, w)
	}
	return out
}

type Count struct {
	Word string
	N    int
}

func (t *Tokenizer) Count(s string) []Count {
	m := map[string]int{}
	for _, w := range t.Tokens(s) {
		m[w]++
	}
	out := make([]Count, 0, len(m))
	for k, v := range m {
		out = append(out, Count{Word: k, N: v})
	}
	// 次数相同时按词排，输出得稳定
	sort.Slice(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		return out[i].Word < out[j].Word
	})
	return out
}

// 相邻词组合，看常见搭配
func (t *Tokenizer) Ngrams(s string, n int) []Count {
	if n < 1 {
		n = 1
	}
	toks := t.Tokens(s)
	if len(toks) < n {
		return nil
	}

	m := map[string]int{}
	for i := 0; i+n <= len(toks); i++ {
		m[strings.Join(toks[i:i+n], " ")]++
	}
	out := make([]Count, 0, len(m))
	for k, v := range m {
		out = append(out, Count{Word: k, N: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		return out[i].Word < out[j].Word
	})
	return out
}

type Summary struct {
	Chars    int // 字符数，不是字节数
	Bytes    int
	Words    int
	Unique   int
	Lines    int
	CJKChars int
}

func (t *Tokenizer) Summarize(s string) Summary {
	var sm Summary
	sm.Bytes = len(s)
	for _, r := range s {
		sm.Chars++
		if isCJK(r) {
			sm.CJKChars++
		}
	}
	if s != "" {
		sm.Lines = strings.Count(s, "\n") + 1
		// 结尾就是换行的话不该多算一行
		if strings.HasSuffix(s, "\n") {
			sm.Lines--
		}
	}

	toks := t.Tokens(s)
	sm.Words = len(toks)
	uniq := map[string]bool{}
	for _, w := range toks {
		uniq[w] = true
	}
	sm.Unique = len(uniq)
	return sm
}

// 默认的中英文停用词
var DefaultStopwords = []string{
	"the", "a", "an", "and", "or", "but", "of", "to", "in", "on", "at",
	"is", "are", "was", "were", "be", "been", "it", "this", "that",
	"for", "with", "as", "by", "from",
	"的", "了", "和", "是", "在", "我", "有", "就", "不", "人", "都",
	"一", "一个", "上", "也", "很", "到", "说", "要", "去", "你", "会", "着",
}
