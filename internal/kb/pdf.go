package kb

import (
	"bytes"
	"compress/zlib"
	"errors"
	"io"
	"regexp"
	"strings"
)

var (
	streamRe = regexp.MustCompile(`(?s)<<(.*?)>>\s*stream\r?\n(.*?)\r?\nendstream`)
	textRe   = regexp.MustCompile(`(?s)\[(.*?)\]\s*TJ|\((.*?[^\\])\)\s*Tj|T\*|ET`)
	litRe    = regexp.MustCompile(`\(((?:\\.|[^\\)])*)\)`)
)

// pdfText is a small text extractor for PDFs whose content streams use
// simple fonts (most exported reports and documents). PDFs with embedded
// CID fonts or scanned images yield little text; convert those first.
func pdfText(data []byte) (string, error) {
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		return "", errors.New("kb: not a PDF file")
	}
	var b strings.Builder
	for _, m := range streamRe.FindAllSubmatch(data, -1) {
		content := m[2]
		if bytes.Contains(m[1], []byte("FlateDecode")) {
			zr, err := zlib.NewReader(bytes.NewReader(content))
			if err != nil {
				continue
			}
			inflated, err := io.ReadAll(io.LimitReader(zr, 20<<20))
			zr.Close()
			if err != nil && len(inflated) == 0 {
				continue
			}
			content = inflated
		}
		for _, op := range textRe.FindAllSubmatch(content, -1) {
			switch {
			case op[1] != nil:
				for _, lit := range litRe.FindAllSubmatch(op[1], -1) {
					b.WriteString(unescapePDF(string(lit[1])))
				}
			case op[2] != nil:
				b.WriteString(unescapePDF(string(op[2])))
			default:
				b.WriteString("\n")
			}
		}
	}
	text := clean(b.String())
	if text == "" {
		return "", errors.New("kb: no extractable text in PDF (scanned or CID fonts)")
	}
	return text, nil
}

func unescapePDF(s string) string {
	r := strings.NewReplacer(`\(`, "(", `\)`, ")", `\\`, `\`, `\n`, "\n", `\r`, "", `\t`, " ")
	return r.Replace(s)
}
