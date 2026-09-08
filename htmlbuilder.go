package onectechcommon

import (
	"fmt"
	"strings"
)

type HTMLBuilder struct {
	sb strings.Builder
}

func NewHTMLBuilder() *HTMLBuilder {
	return &HTMLBuilder{}
}

func (h *HTMLBuilder) WriteLogo(text string, logo string) {
	h.sb.WriteString(`<div style="display: flex; align-items: center; width: 100%;">`)

	if text != "" {
		h.sb.WriteString(`<span style="font-size: 30px; font-weight: bold; color: #006B9E;">`)
		h.sb.WriteString(text)
		h.sb.WriteString(`</span>`)
	}

	h.sb.WriteString(`<span style="margin-left: auto;">`)
	h.sb.WriteString(logo)
	h.sb.WriteString(`</span>`)

	h.sb.WriteString(`</div>`)
}

func (h *HTMLBuilder) WriteHeader(title string) {
	h.sb.WriteString(`<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>`)
	h.sb.WriteString(title)
	h.sb.WriteString(`</title>
    <style>
        body {
			color: #006B9E;
            margin: 20px;
            padding: 0;
        }
        /* Можно добавить дополнительные стили, если необходимо */
    </style>
</head>
<body>
`)
}

func (h *HTMLBuilder) WriteBodyTitleH1(headerText string) {
	fmt.Fprintf(&h.sb, "<h1>%s</h1>\n", headerText)
}

func (h *HTMLBuilder) WriteBodyTitleH2(headerText string) {
	fmt.Fprintf(&h.sb, "<h2>%s</h2>\n", headerText)
}

func (h *HTMLBuilder) WriteBodyTitleH3(headerText string) {
	fmt.Fprintf(&h.sb, "<h3>%s</h3>\n", headerText)
}

func (h *HTMLBuilder) WriteBodyTitleH4(headerText string) {
	fmt.Fprintf(&h.sb, "<h4>%s</h4>\n", headerText)
}

func (h *HTMLBuilder) WriteContent(content string) {
	h.sb.WriteString(content)
}

func (h *HTMLBuilder) WriteFooter() {
	h.sb.WriteString("\n</body>\n</html>")
}

func (h *HTMLBuilder) String() string {
	return h.sb.String()
}
