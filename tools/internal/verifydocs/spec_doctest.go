package verifydocs

import (
	"fmt"
	"os"
	"strings"

	"tetra_language/compiler"
)

func verifySpecCodeBlocks(paths []string) error {
	var errs []string
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		blocks, err := extractSpecCodeBlocks(string(raw))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		for i, block := range blocks {
			if block.skip {
				continue
			}
			filename := fmt.Sprintf("%s#spec%d", path, i+1)
			if _, err := compiler.ParseFile([]byte(block.body), filename); err != nil {
				errs = append(errs, fmt.Sprintf("%s spec block %d parse: %v", path, i+1, err))
				continue
			}
			if !block.check {
				continue
			}
			prog, err := compiler.Parse([]byte(block.body))
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s spec block %d check setup: %v", path, i+1, err))
				continue
			}
			if _, err := compiler.Check(prog); err != nil {
				errs = append(errs, fmt.Sprintf("%s spec block %d check: %v", path, i+1, err))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func extractSpecCodeBlocks(doc string) ([]specCodeBlock, error) {
	var blocks []specCodeBlock
	lines := strings.Split(doc, "\n")
	inBlock := false
	var current []string
	var block specCodeBlock
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			lang, info, ok := specCodeFenceInfo(trimmed)
			if !ok {
				continue
			}
			inBlock = true
			current = nil
			block = specCodeBlock{
				lang:      lang,
				info:      info,
				startLine: i + 1,
				check:     specCodeBlockHasTag(info, "check"),
				skip:      specCodeBlockSkipped(info),
			}
			continue
		}
		if trimmed == "```" {
			block.body = strings.Join(current, "\n") + "\n"
			blocks = append(blocks, block)
			inBlock = false
			current = nil
			block = specCodeBlock{}
			continue
		}
		current = append(current, line)
	}
	if inBlock {
		return nil, fmt.Errorf(
			"unterminated %s spec block starting at line %d",
			block.lang,
			block.startLine,
		)
	}
	return blocks, nil
}

func specCodeFenceInfo(trimmed string) (lang string, info string, ok bool) {
	if !strings.HasPrefix(trimmed, "```") {
		return "", "", false
	}
	info = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
	if info == "" {
		return "", "", false
	}
	fields := strings.Fields(info)
	if len(fields) == 0 {
		return "", "", false
	}
	lang = strings.ToLower(fields[0])
	if lang != "tetra" && lang != "t4" {
		return "", "", false
	}
	return lang, strings.ToLower(info), true
}

func specCodeBlockSkipped(info string) bool {
	for _, tag := range []string{
		"pseudocode",
		"negative",
		"unsupported",
		"skip",
		"noverify",
		"no-verify",
	} {
		if specCodeBlockHasTag(info, tag) {
			return true
		}
	}
	return false
}

func specCodeBlockHasTag(info string, tag string) bool {
	for _, field := range strings.Fields(strings.ToLower(info)) {
		if field == tag {
			return true
		}
	}
	return false
}

func extractTetraDoctests(doc string) ([]string, error) {
	var blocks []string
	lines := strings.Split(doc, "\n")
	inBlock := false
	commentBlock := false
	var current []string
	startLine := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		commentLine, hasCommentPrefix := stripLineCommentPrefix(line)
		commentTrimmed := strings.TrimSpace(commentLine)
		if !inBlock {
			switch {
			case trimmed == "```tetra doctest":
				inBlock = true
				commentBlock = false
				current = nil
				startLine = i + 1
			case hasCommentPrefix && commentTrimmed == "```tetra doctest":
				inBlock = true
				commentBlock = true
				current = nil
				startLine = i + 1
			}
			continue
		}
		if (!commentBlock && trimmed == "```") ||
			(commentBlock && hasCommentPrefix && commentTrimmed == "```") {
			blocks = append(blocks, strings.Join(current, "\n")+"\n")
			inBlock = false
			commentBlock = false
			current = nil
			startLine = 0
			continue
		}
		if commentBlock {
			if !hasCommentPrefix {
				return nil, fmt.Errorf(
					"non-comment line in tetra doctest block starting at line %d",
					startLine,
				)
			}
			current = append(current, commentLine)
			continue
		}
		current = append(current, line)
	}
	if inBlock {
		return nil, fmt.Errorf("unterminated tetra doctest block starting at line %d", startLine)
	}
	return blocks, nil
}

func stripLineCommentPrefix(line string) (string, bool) {
	trimmedLeft := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmedLeft, "//") {
		return "", false
	}
	afterPrefix := strings.TrimPrefix(trimmedLeft, "//")
	if strings.HasPrefix(afterPrefix, " ") {
		afterPrefix = strings.TrimPrefix(afterPrefix, " ")
	}
	return afterPrefix, true
}
