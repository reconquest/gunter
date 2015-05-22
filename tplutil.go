package main

import "regexp"

var reTplEndTagNewLineFix = regexp.MustCompile(
	`(?s)\n?([ \t]+)?{{\s+?\-\s+?end\s+?}}`,
)

func tplStripWhitespaces(template string) string {
	template = reTplEndTagNewLineFix.ReplaceAllString(
		template, "{{ end }}",
	)

	return template
}
