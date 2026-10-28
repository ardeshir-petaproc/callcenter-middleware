// Package kb manages each tenant's knowledge base: encrypted file storage,
// text extraction, full-text indexing, and permission-aware search for
// employees and the AI agent.
package kb

import (
	"archive/zip"
	"bytes"
	"errors"
	"html"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// ErrUnsupported is returned for file types without a text extractor.
var ErrUnsupported = errors.New("kb: unsupported file type")

var (
	tagRe   = regexp.MustCompile(`(?s)<[^>]*>`)
	spaceRe = regexp.MustCompile(`[ \t\r\f\v]+`)
	nlRe    = regexp.MustCompile(`\n{3,}`)
)

// Extract returns the plain text of a document.
func Extract(name string, data []byte) (string, error) {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".txt", ".md", ".markdown", ".csv", ".tsv", ".json", ".log", ".yaml", ".yml", "":
		if !utf8.Valid(data) {
			return "", errors.New("kb: text file is not valid UTF-8")
		}
		return clean(string(data)), nil
	case ".html", ".htm":
		return stripMarkup(string(data)), nil
	case ".docx":
		return officeText(data, func(n string) bool { return n == "word/document.xml" }, "</w:p>")
	case ".pptx":
		return officeText(data, func(n string) bool { return strings.HasPrefix(n, "ppt/slides/slide") }, "</a:p>")
	case ".xlsx":
		return officeText(data, func(n string) bool { return n == "xl/sharedStrings.xml" }, "</si>")
	case ".pdf":
		return pdfText(data)
	}
	return "", ErrUnsupported
}

func clean(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(nlRe.ReplaceAllString(s, "\n\n"))
}

func stripMarkup(s string) string {
	s = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`(?i)<(br|/p|/div|/li|/h[1-6]|/tr)[^>]*>`).ReplaceAllString(s, "\n")
	return clean(html.UnescapeString(tagRe.ReplaceAllString(s, " ")))
}

// officeText reads matching XML parts of an OOXML package and turns the
// given closing tag into paragraph breaks.
func officeText(data []byte, want func(string) bool, paraEnd string) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", errors.New("kb: not a valid Office document")
	}
	var names []string
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		if want(f.Name) {
			names = append(names, f.Name)
			files[f.Name] = f
		}
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		rc, err := files[n].Open()
		if err != nil {
			return "", err
		}
		raw, err := io.ReadAll(io.LimitReader(rc, 50<<20))
		rc.Close()
		if err != nil {
			return "", err
		}
		x := strings.ReplaceAll(string(raw), paraEnd, "\n")
		b.WriteString(html.UnescapeString(tagRe.ReplaceAllString(x, "")))
		b.WriteString("\n")
	}
	return clean(b.String()), nil
}

// Chunk splits text into passages of about size characters on paragraph
// boundaries, for indexing and for quoting to the AI.
func Chunk(text string, size int) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" {
			out = append(out, s)
		}
		cur.Reset()
	}
	for _, para := range strings.Split(text, "\n") {
		for len([]rune(para)) > size {
			r := []rune(para)
			flush()
			cur.WriteString(string(r[:size]))
			flush()
			para = string(r[size:])
		}
		if cur.Len()+len(para) > size {
			flush()
		}
		cur.WriteString(para)
		cur.WriteString("\n")
	}
	flush()
	return out
}
