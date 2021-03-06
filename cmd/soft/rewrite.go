package main

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var printerCfg = &printer.Config{
	Tabwidth: 4,
	Mode:     printer.SourcePos,
}

// TODO: make sure that "soft" is not used and handle case when "atomic" is imported under a different name
func addSoftImport(fset *token.FileSet, f *ast.File) {
	importSpecs := []ast.Spec{
		&ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: `"soft"`,
			},
		},
		&ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: `"sync/atomic"`,
			},
		},
	}

	alreadyImported := make(map[string]bool)

	for _, d := range f.Decls {
		if d, ok := d.(*ast.GenDecl); ok && d.Tok == token.IMPORT {
			for _, sp := range d.Specs {
				if sp, ok := sp.(*ast.ImportSpec); ok {
					// we need to replace github import because "soft" and "golang-soft-mocks" are treated as different packages
					if sp.Path.Value == `"github.com/YuriyNasretdinov/golang-soft-mocks"` {
						sp.Path.Value = `"soft"`
					}
					alreadyImported[sp.Path.Value] = true
				}
			}
		}
	}

	var specs []ast.Spec

	for _, sp := range importSpecs {
		if alreadyImported[sp.(*ast.ImportSpec).Path.Value] {
			continue
		}
		specs = append(specs, sp)
	}

	if len(specs) == 0 {
		return
	}

	decls := make([]ast.Decl, 0, len(f.Decls)+len(specs))
	for _, sp := range specs {
		decls = append(decls, &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: []ast.Spec{sp},
		})
	}

	decls = append(decls, f.Decls...)

	f.Decls = decls
}

func funcDeclFlagName(fset *token.FileSet, d *ast.FuncDecl) string {
	var parts []string
	if d.Body == nil {
		return "" // no body, so obviously cannot mock it
	}
	parts = append(parts, fset.Position(d.Body.Lbrace).String(), fset.Position(d.Body.Rbrace).String())
	h := md5.New()
	for _, p := range parts {
		h.Write([]byte(p))
	}
	return fmt.Sprintf("softMocksFlag_%x", h.Sum(nil))
}

// checks if we have situation like "func (file *file) close() error" in "os" package
// TODO: we can actually rename arguments when this happens so there is no ambiguity
func typesClashWithArgNames(decls []*ast.Field) bool {
	namesMap := make(map[string]bool)
	for _, d := range decls {
		for _, n := range d.Names {
			namesMap[n.Name] = true
		}
	}

	clash := false

	for _, d := range decls {
		ast.Inspect(d.Type, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Ident:
				if namesMap[n.Name] {
					clash = true
				}
			}
			return true
		})
	}

	return clash
}

func funcDeclExpr(f *ast.FuncDecl) ast.Expr {
	if f.Recv == nil {
		return ast.NewIdent(f.Name.Name)
	}

	return &ast.SelectorExpr{
		X:   &ast.ParenExpr{X: f.Recv.List[0].Type},
		Sel: ast.NewIdent(f.Name.Name),
	}
}

var ErrNoNames = errors.New("No names in receiver")

func argNamesFromFuncDecl(f *ast.FuncDecl) ([]ast.Expr, bool, error) {
	var res []ast.Expr
	var haveEllipsis bool

	if f.Recv != nil {
		names := f.Recv.List[0].Names
		if len(names) == 0 {
			return nil, false, ErrNoNames
		}
		res = append(res, names[0])
	}

	for _, t := range f.Type.Params.List {
		if len(t.Names) == 0 {
			return nil, false, ErrNoNames
		}

		for _, n := range t.Names {
			if _, ok := t.Type.(*ast.Ellipsis); ok {
				haveEllipsis = true
			}
			res = append(res, n)
		}
	}

	return res, haveEllipsis, nil
}

func funcDeclType(f *ast.FuncDecl) ast.Expr {
	var in []*ast.Field

	if f.Recv != nil {
		in = append(in, f.Recv.List[0])
	}

	for _, t := range f.Type.Params.List {
		in = append(in, t)
	}

	if typesClashWithArgNames(in) {
		return nil
	}

	return &ast.FuncType{
		Params:  &ast.FieldList{List: in},
		Results: f.Type.Results,
	}
}

type funcFlags map[*ast.FuncDecl]string

func addInit(hashes funcFlags, initFunc *ast.FuncDecl, fset *token.FileSet, f *ast.File) {
	specs := &ast.ValueSpec{
		Type: ast.NewIdent("int32"),
	}

	for decl, flagName := range hashes {
		specs.Names = append(specs.Names, ast.NewIdent(flagName))

		initFunc.Body.List = append(initFunc.Body.List, &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("soft"),
					Sel: ast.NewIdent("RegisterFunc"),
				},
				Args: []ast.Expr{
					funcDeclExpr(decl),
					&ast.UnaryExpr{
						Op: token.AND,
						X:  ast.NewIdent(flagName),
					},
				},
			},
		})

	}

	f.Decls = append(f.Decls, &ast.GenDecl{
		Tok:   token.VAR,
		Specs: []ast.Spec{specs},
	})
}

func getInterceptorsExpression(decl *ast.FuncDecl) ast.Expr {
	funcType := funcDeclType(decl)
	if funcType == nil {
		return nil
	}

	args, haveEllipsis, err := argNamesFromFuncDecl(decl)
	if err != nil {
		return nil
	}

	expr := &ast.CallExpr{
		Fun: &ast.TypeAssertExpr{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("soft"),
					Sel: ast.NewIdent("GetMockFor"),
				},
				Args: []ast.Expr{funcDeclExpr(decl)},
			},
			Type: funcType,
		},
		Args: args,
	}

	if haveEllipsis {
		expr.Ellipsis = 1
	}

	return expr
}

func injectInterceptors(flags funcFlags) {
	for decl, flagName := range flags {
		var injectBody []ast.Stmt
		interceptorsExpr := getInterceptorsExpression(decl)
		if interceptorsExpr == nil {
			delete(flags, decl)
			continue
		}

		if decl.Type.Results != nil {
			injectBody = append(injectBody, &ast.ReturnStmt{Results: []ast.Expr{interceptorsExpr}})
		} else {
			injectBody = append(injectBody, &ast.ExprStmt{X: interceptorsExpr}, &ast.ReturnStmt{})
		}

		newList := make([]ast.Stmt, 0, len(decl.Body.List)+1)
		newList = append(newList, &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				Op: token.NEQ,
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("atomic"),
						Sel: ast.NewIdent("LoadInt32"),
					},
					Args: []ast.Expr{&ast.UnaryExpr{
						Op: token.AND,
						X:  ast.NewIdent(flagName),
					}},
				},
				Y: &ast.BasicLit{
					Kind:  token.INT,
					Value: "0",
				},
			},
			Body: &ast.BlockStmt{List: injectBody},
		})
		newList = append(newList, decl.Body.List...)
		decl.Body.List = newList
	}
}

func transformAst(fset *token.FileSet, f *ast.File) {
	flags := make(funcFlags)
	var initFunc *ast.FuncDecl

	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Name.Name == "init" && d.Recv == nil {
				initFunc = d
			} else if flName := funcDeclFlagName(fset, d); flName != "" {
				flags[d] = flName
			}
		}
	}

	injectInterceptors(flags)

	if len(flags) == 0 {
		return
	}

	addSoftImport(fset, f)

	if initFunc == nil {
		initFunc = &ast.FuncDecl{
			Name: ast.NewIdent("init"),
			Type: &ast.FuncType{},
			Body: &ast.BlockStmt{},
		}

		f.Decls = append(f.Decls, initFunc)
	}

	addInit(flags, initFunc, fset, f)
}

func isPackage(pkg, filename string) bool {
	return strings.HasPrefix(filename, filepath.Join(goroot, "src", filepath.FromSlash(pkg))+string(os.PathSeparator))
}

// checks only exact package, not subpackages (because examples and the soft util itself live there)
func isSoftPackage(filename string) bool {
	return filepath.Dir(filename) == filepath.Join(gopath, "src", "github.com", "YuriyNasretdinov", "golang-soft-mocks")
}

// These packages are used by soft mocks themselves so otherwise we would get cyclic imports
var excludedPackages = []string{
	"sync/atomic",
	"sync",
	"reflect",
	"soft",
	"runtime",
	"math",
	"unsafe",
	"strconv",
	"internal",
	"errors",
	"unicode/utf8",
}

func rewriteFile(filename string) (contents []byte, err error) {
	if !strings.HasSuffix(filename, ".go") || isSoftPackage(filename) {
		return ioutil.ReadFile(filename)
	}

	for _, pkg := range excludedPackages {
		if isPackage(pkg, filename) {
			return ioutil.ReadFile(filename)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	cmap := ast.NewCommentMap(fset, f, f.Comments)
	transformAst(fset, f)
	f.Comments = cmap.Filter(f).Comments()

	var b bytes.Buffer
	if err := printerCfg.Fprint(&b, fset, f); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
