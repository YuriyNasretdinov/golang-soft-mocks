package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"soft"
)

var printerCfg = &printer.Config{
	Tabwidth: 4,
	Mode:     printer.SourcePos,
}

// TODO: make better code for situation when there is not enough disk space
func createBackupFile(filename string) error {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	fp, err := os.OpenFile(filename+".bak", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)

	if err != nil {
		if os.IsExist(err) {
			return nil
		}

		return err
	}

	defer fp.Close()

	if _, err := fp.Write(contents); err != nil {
		return err
	}

	return nil
}

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

	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				d.Specs = append(d.Specs, importSpecs...)
				return
			}
		}
	}

	f.Decls = append(f.Decls, &ast.GenDecl{
		Tok:   token.IMPORT,
		Specs: importSpecs,
	})
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

func argNamesFromFuncDecl(f *ast.FuncDecl) []ast.Expr {
	var res []ast.Expr

	if f.Recv != nil {
		res = append(res, f.Recv.List[0].Names[0])
	}

	for _, t := range f.Type.Params.List {
		for _, n := range t.Names {
			res = append(res, n)
		}
	}

	return res
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

	return &ast.CallExpr{
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
		Args: argNamesFromFuncDecl(decl),
	}
}

func injectInterceptors(flags funcFlags) {
	for decl, flagName := range flags {
		var injectBody []ast.Stmt
		interceptorsExpr := getInterceptorsExpression(decl)
		if interceptorsExpr == nil {
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
	addSoftImport(fset, f)

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

	if initFunc == nil {
		initFunc = &ast.FuncDecl{
			Name: ast.NewIdent("init"),
			Type: &ast.FuncType{},
			Body: &ast.BlockStmt{},
		}

		f.Decls = append(f.Decls, initFunc)
	}

	addInit(flags, initFunc, fset, f)
	injectInterceptors(flags)
}

func rewriteFile(filename string) error {
	if err := createBackupFile(filename); err != nil {
		return err
	}

	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return err
	}

	transformAst(fset, f)

	var b bytes.Buffer
	if err := printerCfg.Fprint(&b, fset, f); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filename, b.Bytes(), 0666); err != nil {
		return err
	}

	return nil
}

func main() {
	filename := filepath.Join(runtime.GOROOT(), "src", "os", "file_unix.go")

	if err := rewriteFile(filename); err != nil {
		log.Printf("Could not rewrite %s: %s", filename, err)
		return
	}

	closeFunc := (*os.File).Close
	soft.Mock(closeFunc, func(f *os.File) error {
		fmt.Printf("File is going to be closed: %s\n", f.Name())
		res, _ := soft.CallOriginal(closeFunc, f)[0].(error)
		return res
	})

	fp, _ := os.Open("/dev/null")
	err := fp.Close()

	fmt.Println("Hello, world: %v!", err)
}
