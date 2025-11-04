package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxLinesDefault = 400

type importInfo struct {
	nameKey string
	order   int
	text    string
}

type declInfo struct {
	decl        ast.Decl
	startOffset int
	endOffset   int
	lineCount   int
	name        string
}

type declGroup struct {
	decls []*declInfo
}

func main() {
	root := flag.String("dir", ".", "directory to scan")
	maxLines := flag.Int("max", maxLinesDefault, "maximum lines per file")
	flag.Parse()

	files, err := collectLargeFiles(*root, *maxLines)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error collecting files: %v\n", err)
		os.Exit(1)
	}

	for _, path := range files {
		if err := splitFile(path, *maxLines); err != nil {
			fmt.Fprintf(os.Stderr, "failed to split %s: %v\n", path, err)
			os.Exit(1)
		}
	}
}

func collectLargeFiles(root string, maxLines int) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		lines, err := countLines(path)
		if err != nil {
			return err
		}
		if lines > maxLines {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	buf := make([]byte, 32*1024)
	lines := 0
	for {
		n, err := f.Read(buf)
		lines += bytes.Count(buf[:n], []byte{'\n'})
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}
	return lines, nil
}

func splitFile(path string, maxLines int) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, path, src, parser.ParseComments)
	if err != nil {
		return err
	}

	imports, importMap := gatherImports(file, src, set)
	decls := gatherDecls(file, src, set)
	if len(decls) == 0 {
		return nil
	}

	headerFirst, headerOther := buildHeaders(file, src, set)
	groups := initialGroups(decls, maxLines)

	contents, err := materializeGroups(path, groups, headerFirst, headerOther, imports, importMap, src, maxLines)
	if err != nil {
		return err
	}

	for i, content := range contents {
		target := path
		if i > 0 {
			var writeErr error
			target, writeErr = partPath(path, i+1)
			if writeErr != nil {
				return writeErr
			}
		}
		if err := os.WriteFile(target, content, 0o644); err != nil {
			return err
		}
	}

	return nil
}

func gatherImports(file *ast.File, src []byte, set *token.FileSet) ([]*importInfo, map[string]*importInfo) {
	var infos []*importInfo
	nameMap := make(map[string]*importInfo)
	order := 0
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		for _, spec := range gen.Specs {
			imp, ok := spec.(*ast.ImportSpec)
			if !ok {
				continue
			}
			info := &importInfo{
				nameKey: importName(imp),
				order:   order,
				text:    strings.TrimSpace(specText(imp, src, set)),
			}
			infos = append(infos, info)
			nameMap[info.nameKey] = info
			order++
		}
	}
	return infos, nameMap
}

func importName(spec *ast.ImportSpec) string {
	if spec.Name != nil {
		return spec.Name.Name
	}
	path := strings.Trim(spec.Path.Value, "\"")
	return filepath.Base(path)
}

func specText(spec *ast.ImportSpec, src []byte, set *token.FileSet) string {
	start := set.Position(spec.Pos()).Offset
	end := set.Position(spec.End()).Offset
	if end > len(src) {
		end = len(src)
	}
	return string(src[start:end])
}

func gatherDecls(file *ast.File, src []byte, set *token.FileSet) []*declInfo {
	var decls []*declInfo
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			continue
		}
		startPos := set.Position(decl.Pos())
		endPos := set.Position(decl.End())
		info := &declInfo{
			decl:        decl,
			startOffset: startPos.Offset,
			endOffset:   endPos.Offset,
			lineCount:   endPos.Line - startPos.Line + 1,
			name:        declName(decl),
		}
		if info.endOffset > len(src) {
			info.endOffset = len(src)
		}
		decls = append(decls, info)
	}
	return decls
}

func declName(decl ast.Decl) string {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		return d.Name.Name
	case *ast.GenDecl:
		if len(d.Specs) > 0 {
			switch spec := d.Specs[0].(type) {
			case *ast.TypeSpec:
				return spec.Name.Name
			case *ast.ValueSpec:
				if len(spec.Names) > 0 {
					return spec.Names[0].Name
				}
			}
		}
		return d.Tok.String()
	default:
		return "declaration"
	}
}

func buildHeaders(file *ast.File, src []byte, set *token.FileSet) (string, string) {
	pkgOffset := set.Position(file.Package).Offset
	lineEnd := pkgOffset
	for lineEnd < len(src) && src[lineEnd] != '\n' {
		lineEnd++
	}
	if lineEnd < len(src) {
		lineEnd++
	}
	prefix := string(src[:pkgOffset])
	packageLine := string(src[pkgOffset:lineEnd])

	first := prefix + packageLine
	if !strings.HasSuffix(first, "\n") {
		first += "\n"
	}
	other := packageLine
	if !strings.HasSuffix(other, "\n") {
		other += "\n"
	}
	return first, other
}

func initialGroups(decls []*declInfo, maxLines int) []*declGroup {
	if len(decls) == 0 {
		return nil
	}
	var groups []*declGroup
	current := &declGroup{}
	maxGroupLines := maxLines - 40
	if maxGroupLines < 100 {
		maxGroupLines = maxLines - 10
	}
	lineSum := 0
	for _, decl := range decls {
		if len(current.decls) > 0 && lineSum+decl.lineCount > maxGroupLines {
			groups = append(groups, current)
			current = &declGroup{}
			lineSum = 0
		}
		current.decls = append(current.decls, decl)
		lineSum += decl.lineCount
	}
	groups = append(groups, current)
	return groups
}

func materializeGroups(path string, groups []*declGroup, headerFirst, headerOther string, imports []*importInfo, importMap map[string]*importInfo, src []byte, maxLines int) ([][]byte, error) {
	var contents [][]byte
	for i := 0; i < len(groups); i++ {
		header := headerOther
		if i == 0 {
			header = headerFirst
		}
		content, lineCount, err := buildGroupContent(groups[i], header, imports, importMap, src)
		if err != nil {
			return nil, err
		}
		if lineCount > maxLines && len(groups[i].decls) > 1 {
			last := groups[i].decls[len(groups[i].decls)-1]
			groups[i].decls = groups[i].decls[:len(groups[i].decls)-1]
			if i+1 >= len(groups) {
				groups = append(groups, &declGroup{})
			}
			next := groups[i+1]
			next.decls = append([]*declInfo{last}, next.decls...)
			// retry current group with adjusted declarations
			i--
			contents = contents[:len(contents)]
			continue
		}
		if lineCount > maxLines {
			return nil, fmt.Errorf("declaration %q in %s still exceeds %d lines", groups[i].decls[0].name, path, maxLines)
		}
		contents = append(contents, content)
	}
	return contents, nil
}

func buildGroupContent(group *declGroup, header string, imports []*importInfo, importMap map[string]*importInfo, src []byte) ([]byte, int, error) {
	used := make(map[*importInfo]bool)
	for _, decl := range group.decls {
		ast.Inspect(decl.decl, func(n ast.Node) bool {
			ident, ok := n.(*ast.Ident)
			if !ok || ident.Obj != nil {
				return true
			}
			if info, ok := importMap[ident.Name]; ok {
				used[info] = true
			}
			return true
		})
	}
	var importLines []string
	for _, info := range imports {
		if used[info] {
			line := strings.TrimSpace(info.text)
			importLines = append(importLines, "\t"+line)
		}
	}

	var builder strings.Builder
	builder.WriteString(strings.TrimRight(header, "\n"))
	builder.WriteString("\n\n")
	if len(importLines) > 0 {
		builder.WriteString("import (\n")
		for _, line := range importLines {
			builder.WriteString(line)
			if !strings.HasSuffix(line, "\n") {
				builder.WriteString("\n")
			}
		}
		builder.WriteString(")\n\n")
	}
	for idx, decl := range group.decls {
		snippet := string(src[decl.startOffset:decl.endOffset])
		s := strings.TrimLeft(snippet, "\n")
		builder.WriteString(s)
		if !strings.HasSuffix(s, "\n") {
			builder.WriteString("\n")
		}
		if idx < len(group.decls)-1 {
			builder.WriteString("\n")
		}
	}

	formatted, err := format.Source([]byte(builder.String()))
	if err != nil {
		return nil, 0, err
	}
	lineCount := bytes.Count(formatted, []byte{'\n'})
	if len(formatted) == 0 || formatted[len(formatted)-1] != '\n' {
		lineCount++
	}
	return formatted, lineCount, nil
}

func partPath(path string, index int) (string, error) {
	dir := filepath.Dir(path)
	base := strings.TrimSuffix(filepath.Base(path), ".go")
	candidate := filepath.Join(dir, fmt.Sprintf("%s_part%d.go", base, index))
	if _, err := os.Stat(candidate); errors.Is(err, fs.ErrNotExist) {
		return candidate, nil
	}
	return candidate, nil
}
