package natsworkout

import (
	"bufio"
	"errors"
	"os"
)

type Words []string

type WordsCursor struct {
	words  Words
	cursor int
}

func isAlpha(w string) bool {
	for _, r := range w {
		if r < 'a' || r > 'z' {
			return false
		}
	}

	return true
}

func OpenWords() (*Words, error) {
	f, err := os.Open("/usr/share/dict/words")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var words []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		word := scanner.Text()
		if isAlpha(word) {
			words = append(words, word)
		}
	}

	if scanner.Err() != nil {
		return nil, err
	}

	if len(words) == 0 {
		return nil, errors.New("no words")
	}

	return (*Words)(&words), nil
}

func (w Words) Cursor() *WordsCursor {
	return &WordsCursor{words: w}
}

func (wc *WordsCursor) Next() string {
	w := wc.words[wc.cursor]
	wc.cursor++
	if wc.cursor >= len(wc.words) {
		wc.cursor = 0
	}
	return w
}
