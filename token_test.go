package main

import (
	"reflect"
	"strings"
	"testing"
)

func tk(opt Options) *Tokenizer { return NewTokenizer(opt) }

func TestSplitEnglish(t *testing.T) {
	got := tk(Options{}).Split("Hello, world! Go is fun.")
	want := []string{"Hello", "world", "Go", "is", "fun"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("切词不对: %v", got)
	}
}

// don't 和 well-known 里的符号是词的一部分，不能拆开
func TestSplitKeepsInnerPunct(t *testing.T) {
	got := tk(Options{}).Split("don't split well-known words")
	want := []string{"don't", "split", "well-known", "words"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("词内符号被拆了: %v", got)
	}
}

// 但结尾的撇号和连字符不该粘上去
func TestSplitTrailingPunct(t *testing.T) {
	got := tk(Options{}).Split("boys' toys - here")
	for _, w := range got {
		if strings.HasSuffix(w, "'") || strings.HasSuffix(w, "-") {
			t.Errorf("结尾的符号该被切掉: %q（全部 %v）", w, got)
		}
	}
}

func TestSplitCJKSingle(t *testing.T) {
	got := tk(Options{}).Split("中文分词")
	want := []string{"中", "文", "分", "词"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("单字切分不对: %v", got)
	}
}

func TestSplitCJKBigram(t *testing.T) {
	got := tk(Options{CJKGram: 2}).Split("中文分词")
	want := []string{"中文", "文分", "分词"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("双字切分不对: %v", got)
	}
}

// 整段比 gram 还短时不能返回空
func TestCJKShorterThanGram(t *testing.T) {
	got := tk(Options{CJKGram: 3}).Split("中文")
	if len(got) == 0 {
		t.Fatal("短于 gram 长度的不该被吞掉")
	}
	if got[0] != "中文" {
		t.Errorf("该整段当一个词，得到 %v", got)
	}
}

// 中英文混排要各切各的
func TestSplitMixed(t *testing.T) {
	got := tk(Options{}).Split("用 Go 写 CLI 工具")
	want := []string{"用", "Go", "写", "CLI", "工", "具"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("混排切词不对: %v", got)
	}
}

func TestSplitDigitsAndUnderscore(t *testing.T) {
	got := tk(Options{}).Split("go1.21 my_var 42")
	// 小数点不是词的一部分，会被切开
	if len(got) < 3 {
		t.Errorf("数字和下划线该被保留: %v", got)
	}
	found := false
	for _, w := range got {
		if w == "my_var" {
			found = true
		}
	}
	if !found {
		t.Errorf("下划线该算词内字符: %v", got)
	}
}

func TestSplitEmpty(t *testing.T) {
	if got := tk(Options{}).Split(""); len(got) != 0 {
		t.Errorf("空串该切出空，得到 %v", got)
	}
	if got := tk(Options{}).Split("!!!   ,,,"); len(got) != 0 {
		t.Errorf("纯标点该切出空，得到 %v", got)
	}
}

func TestLower(t *testing.T) {
	got := tk(Options{Lower: true}).Tokens("Go GO go")
	for _, w := range got {
		if w != "go" {
			t.Errorf("该全转小写，得到 %q", w)
		}
	}
}

// MinLen 得按字符数算，按字节数的话中文全被滤掉了
func TestMinLenCountsRunes(t *testing.T) {
	got := tk(Options{MinLen: 2}).Tokens("中文")
	if len(got) != 0 {
		// 单字切分下每个词只有 1 个字符，该被 min=2 滤掉
		t.Errorf("单字该被 min=2 滤掉，得到 %v", got)
	}
	// 双字切分下就该留下
	got = tk(Options{MinLen: 2, CJKGram: 2}).Tokens("中文分词")
	if len(got) != 3 {
		t.Errorf("双字词不该被滤掉，得到 %v", got)
	}
}

func TestMinLenEnglish(t *testing.T) {
	got := tk(Options{MinLen: 3}).Tokens("a an and ants")
	want := []string{"and", "ants"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("短词该被滤掉: %v", got)
	}
}

func TestStopwords(t *testing.T) {
	got := tk(Options{Stopwords: []string{"the", "is"}}).Tokens("the cat is here")
	want := []string{"cat", "here"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("停用词没滤干净: %v", got)
	}
}

// 开了 Lower 的话停用词也得按小写匹配，否则 "The" 滤不掉
func TestStopwordsWithLower(t *testing.T) {
	got := tk(Options{Lower: true, Stopwords: []string{"The"}}).Tokens("The cat")
	if len(got) != 1 || got[0] != "cat" {
		t.Errorf("大小写不一致导致停用词失效: %v", got)
	}
}

func TestStopwordsIgnoresBlank(t *testing.T) {
	// 空的停用词条目不该把所有词都滤掉
	got := tk(Options{Stopwords: []string{"", "  ", "cat"}}).Tokens("the cat here")
	if len(got) != 2 {
		t.Errorf("空停用词干扰了过滤: %v", got)
	}
}

func TestCount(t *testing.T) {
	got := tk(Options{Lower: true}).Count("go go go rust rust python")
	if got[0].Word != "go" || got[0].N != 3 {
		t.Errorf("第一名该是 go×3，得到 %+v", got[0])
	}
	if got[1].Word != "rust" || got[1].N != 2 {
		t.Errorf("第二名该是 rust×2，得到 %+v", got[1])
	}
}

// 次数相同时按词排，多跑几次结果得一致
func TestCountStableOrder(t *testing.T) {
	text := "b a c"
	first := tk(Options{}).Count(text)
	for i := 0; i < 20; i++ {
		got := tk(Options{}).Count(text)
		if !reflect.DeepEqual(got, first) {
			t.Fatalf("同次数时排序不稳定: %v vs %v", got, first)
		}
	}
	if first[0].Word != "a" {
		t.Errorf("同次数该按字母排，第一个是 %q", first[0].Word)
	}
}

func TestNgrams(t *testing.T) {
	got := tk(Options{Lower: true}).Ngrams("go is fun go is fun go", 2)
	// 三个二元组都出现 2 次，同次数按词排，fun go 排第一
	if got[0].Word != "fun go" || got[0].N != 2 {
		t.Errorf("最常见的二元组该是 fun go×2，得到 %+v", got[0])
	}
	// go is 也得算到 2 次
	found := false
	for _, c := range got {
		if c.Word == "go is" && c.N == 2 {
			found = true
		}
	}
	if !found {
		t.Errorf("go is 该出现 2 次: %+v", got)
	}
}

// 词数不够组成一个 n-gram 时该返回空而不是 panic
func TestNgramsTooShort(t *testing.T) {
	if got := tk(Options{}).Ngrams("hello", 3); len(got) != 0 {
		t.Errorf("词不够该返回空，得到 %v", got)
	}
	if got := tk(Options{}).Ngrams("", 2); len(got) != 0 {
		t.Errorf("空输入该返回空，得到 %v", got)
	}
}

func TestNgramsBadN(t *testing.T) {
	// n 给 0 或负数按 1 处理，不能死循环也不能 panic
	got := tk(Options{}).Ngrams("a b c", 0)
	if len(got) != 3 {
		t.Errorf("n<1 该按 1 处理，得到 %v", got)
	}
}

func TestSummarize(t *testing.T) {
	s := tk(Options{}).Summarize("hello world\nhello go\n")
	if s.Lines != 2 {
		t.Errorf("行数该是 2，得到 %d", s.Lines)
	}
	if s.Words != 4 {
		t.Errorf("词数该是 4，得到 %d", s.Words)
	}
	if s.Unique != 3 {
		t.Errorf("不重复该是 3，得到 %d", s.Unique)
	}
}

// 结尾的换行不该多算一行
func TestSummarizeTrailingNewline(t *testing.T) {
	a := tk(Options{}).Summarize("a\nb")
	b := tk(Options{}).Summarize("a\nb\n")
	if a.Lines != b.Lines {
		t.Errorf("结尾换行不该改变行数: %d vs %d", a.Lines, b.Lines)
	}
}

// 中文的字符数和字节数不是一回事
func TestSummarizeCJKChars(t *testing.T) {
	s := tk(Options{}).Summarize("中文abc")
	if s.Chars != 5 {
		t.Errorf("字符数该是 5，得到 %d", s.Chars)
	}
	if s.Bytes != 9 {
		t.Errorf("字节数该是 9，得到 %d", s.Bytes)
	}
	if s.CJKChars != 2 {
		t.Errorf("中日韩字符该是 2，得到 %d", s.CJKChars)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	s := tk(Options{}).Summarize("")
	if s.Lines != 0 || s.Words != 0 || s.Chars != 0 {
		t.Errorf("空输入统计该全是 0: %+v", s)
	}
}

func TestJapaneseKorean(t *testing.T) {
	got := tk(Options{}).Split("こんにちは안녕")
	if len(got) == 0 {
		t.Fatal("日韩文字没切出来")
	}
	// 假名和谚文都该被当成 CJK 按字切
	for _, w := range got {
		if len([]rune(w)) != 1 {
			t.Errorf("该按单字切: %v", got)
			break
		}
	}
}

func TestCJKGramZeroDefaultsToOne(t *testing.T) {
	x := tk(Options{CJKGram: 0})
	if x.opt.CJKGram != 1 {
		t.Errorf("gram 为 0 该按 1 处理，得到 %d", x.opt.CJKGram)
	}
	x = tk(Options{CJKGram: -5})
	if x.opt.CJKGram != 1 {
		t.Errorf("负 gram 该按 1 处理，得到 %d", x.opt.CJKGram)
	}
}

func TestDefaultStopwordsWork(t *testing.T) {
	got := tk(Options{Lower: true, Stopwords: DefaultStopwords}).Tokens("the quick brown fox")
	for _, w := range got {
		if w == "the" {
			t.Error("默认停用词没生效")
		}
	}
	if len(got) != 3 {
		t.Errorf("该剩 3 个词，得到 %v", got)
	}
}
