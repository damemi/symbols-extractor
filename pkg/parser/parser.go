package parser

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/gofed/symbols-extractor/pkg/parser/alloctable"
	exprparser "github.com/gofed/symbols-extractor/pkg/parser/expression"
	fileparser "github.com/gofed/symbols-extractor/pkg/parser/file"
	stmtparser "github.com/gofed/symbols-extractor/pkg/parser/statement"
	"github.com/gofed/symbols-extractor/pkg/parser/symboltable/global"
	"github.com/gofed/symbols-extractor/pkg/parser/symboltable/stack"
	typeparser "github.com/gofed/symbols-extractor/pkg/parser/type"
	"github.com/gofed/symbols-extractor/pkg/parser/types"
	gotypes "github.com/gofed/symbols-extractor/pkg/types"
)

// Context participants:
// - package (fully qualified package name, e.g. github.com/coreos/etcd/pkg/wait)
// - package file (package + its underlying filename)
// - package symbol definition (AST of a symbol definition)

// FileContext storing context for a file
type FileContext struct {
	// package's underlying filename
	Filename string
	// AST of a file (so the AST is not constructed again once all file's dependencies are processed)
	FileAST *ast.File
	// If set, there are still some symbols that needs processing
	ImportsProcessed bool

	DataTypes []*ast.TypeSpec
	Variables []*ast.ValueSpec
	Functions []*ast.FuncDecl
}

// PackageContext storing context for a package
type PackageContext struct {
	// fully qualified package name
	PackagePath string

	// files attached to a package
	PackageDir string
	FileIndex  int
	Files      []*FileContext

	Config *types.Config

	// package name
	PackageName string
	// per file symbol table
	SymbolTable *stack.Stack
	// per file allocatable ST
	AllocatedSymbolsTable *alloctable.Table

	// symbol definitions postponed (only variable/constants and function bodies definitions affected)
	DataTypes []*ast.TypeSpec
	Variables []*ast.ValueSpec
	Functions []*ast.FuncDecl
}

// Idea:
// - process the input package
// - retrieve all input package files
// - process each file of the input package
// - process a list of imported packages in each file
// - if any of the imported packages is not yet parsed out the package at the top of the package stack
// - pick a package from the top of the package stack
// - repeat the process until all imported packages are processed
// - continue processing declarations/definitions in the file
// - if any of the decls/defs in the file are not processed completely, put it in the postponed list
// - once all files in the package are processed, start re-processing the decls/defs in the postponed list
// - once all decls/defs are processed, clear the package and pick new package from the package stack

type ProjectParser struct {
	packagePath string
	// Global symbol table
	globalSymbolTable *global.Table
	// For each package and its file store its alloc symbol table
	allocSymbolTable map[string]map[string]*alloctable.Table

	// package stack
	packageStack []*PackageContext
}

func New(packagePath string) *ProjectParser {
	return &ProjectParser{
		packagePath:       packagePath,
		packageStack:      make([]*PackageContext, 0),
		globalSymbolTable: global.New(),
	}
}

func (pp *ProjectParser) processImports(imports []*ast.ImportSpec) (missingImports []*gotypes.PackageQualifier) {
	for _, spec := range imports {
		q := fileparser.MakePackageQualifier(spec)
		// Check if the imported package is already processed
		_, err := pp.globalSymbolTable.Lookup(q.Path)
		if err != nil {
			missingImports = append(missingImports, q)
			fmt.Printf("Package %q not yet processed\n", q.Path)
		}
		// TODO(jchaloup): Check if the package is already in the package queue
		//                 If it is it is an error (import cycles are not permitted)
	}
	return
}

func (pp *ProjectParser) getPackageFiles(packagePath string) (files []string, packageLocation string, err error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		return nil, "", fmt.Errorf("GOPATH env not set")
	}

	// TODO(jchaloup): detect the GOROOT env from `go env` command
	goroot := "/usr/lib/golang/"
	godirs := []string{
		path.Join(goroot, "src", packagePath),
		path.Join(gopath, "src", packagePath),
	}
	for _, godir := range godirs {
		fileInfo, err := ioutil.ReadDir(godir)
		if err == nil {
			fmt.Printf("Checking %v...\n", godir)
			for _, file := range fileInfo {
				if !file.Mode().IsRegular() {
					continue
				}
				fmt.Printf("DirFile: %v\n", file.Name())
				// TODO(jchaloup): filter out unacceptable files (only *.go and *.s allowed)
				files = append(files, file.Name())
			}
			fmt.Printf("\n\n")
			return files, godir, nil
		}
	}

	return nil, "", fmt.Errorf("Package %q not found in any of %s locations", packagePath, strings.Join(godirs, ":"))
}

func (pp *ProjectParser) createPackageContext(packagePath string) (*PackageContext, error) {
	c := &PackageContext{
		PackagePath:           packagePath,
		FileIndex:             0,
		SymbolTable:           stack.New(),
		AllocatedSymbolsTable: alloctable.New(),
	}

	files, path, err := pp.getPackageFiles(packagePath)
	if err != nil {
		return nil, err
	}
	c.PackageDir = path
	for _, file := range files {
		c.Files = append(c.Files, &FileContext{Filename: file})
	}

	config := &types.Config{
		PackageName:           packagePath,
		SymbolTable:           c.SymbolTable,
		AllocatedSymbolsTable: c.AllocatedSymbolsTable,
		GlobalSymbolTable:     pp.globalSymbolTable,
	}

	config.TypeParser = typeparser.New(config)
	config.ExprParser = exprparser.New(config)
	config.StmtParser = stmtparser.New(config)

	c.Config = config

	fmt.Printf("PackageContextCreated: %#v\n\n", c)
	return c, nil
}

func (pp *ProjectParser) reprocessDataTypes(p *PackageContext) error {
	fLen := len(p.Files)
	for i := 0; i < fLen; i++ {
		fileContext := p.Files[i]
		if fileContext.DataTypes != nil {
			payload := &fileparser.Payload{
				DataTypes: fileContext.DataTypes,
			}
			fmt.Printf("Types: %#v\n", payload.DataTypes)
			for _, spec := range fileContext.FileAST.Imports {
				payload.Imports = append(payload.Imports, spec)
			}
			if err := fileparser.NewParser(p.Config).Parse(payload); err != nil {
				return err
			}
			fmt.Printf("Types: %#v\n", payload.DataTypes)
			if payload.DataTypes != nil {
				return fmt.Errorf("There are still some postponed data types to process after the second round: %v", p.PackagePath)
			}
		}
	}
	return nil
}

func (pp *ProjectParser) reprocessVariables(p *PackageContext) error {
	fLen := len(p.Files)
	for i := 0; i < fLen; i++ {
		fileContext := p.Files[i]
		if fileContext.Variables != nil {
			payload := &fileparser.Payload{
				Variables: fileContext.Variables,
			}
			fmt.Printf("Vars: %#v\n", payload.Variables)
			for _, spec := range fileContext.FileAST.Imports {
				payload.Imports = append(payload.Imports, spec)
			}
			if err := fileparser.NewParser(p.Config).Parse(payload); err != nil {
				return err
			}
			fmt.Printf("Vars: %#v\n", payload.Variables)
			if payload.Variables != nil {
				return fmt.Errorf("There are still some postponed variables to process after the second round: %v", p.PackagePath)
			}
		}
	}
	return nil
}

func (pp *ProjectParser) reprocessFunctions(p *PackageContext) error {
	fLen := len(p.Files)
	for i := 0; i < fLen; i++ {
		fileContext := p.Files[i]
		if fileContext.Functions != nil {
			payload := &fileparser.Payload{
				Functions: fileContext.Functions,
			}
			fmt.Printf("Funcs: %#v\n", payload.Functions)
			for _, spec := range fileContext.FileAST.Imports {
				payload.Imports = append(payload.Imports, spec)
			}
			if err := fileparser.NewParser(p.Config).Parse(payload); err != nil {
				return err
			}
			fmt.Printf("Funcs: %#v\n", payload.Functions)
			if payload.Functions != nil {
				return fmt.Errorf("There are still some postponed functions to process after the second round: %v", p.PackagePath)
			}
		}
	}
	return nil
}

func (pp *ProjectParser) Parse() error {
	// Process the input package
	c, err := pp.createPackageContext("github.com/gofed/symbols-extractor/pkg/parser/testdata/unordered")
	if err != nil {
		return err
	}
	// Push the input package into the package stack
	pp.packageStack = append(pp.packageStack, c)

PACKAGE_STACK:
	for len(pp.packageStack) > 0 {
		// Process the package stack
		p := pp.packageStack[0]
		fmt.Printf("========PS processing %#v...========\n", p.PackageDir)
		// Process the files
		fLen := len(p.Files)
		for i := p.FileIndex; i < fLen; i++ {
			fileContext := p.Files[i]
			fmt.Printf("fx: %#v\n", fileContext)
			if fileContext.FileAST == nil {
				f, err := parser.ParseFile(token.NewFileSet(), path.Join(p.PackageDir, fileContext.Filename), nil, 0)
				if err != nil {
					return err
				}
				fileContext.FileAST = f
			}
			fmt.Printf("FileAST:\t\t%#v\n", fileContext.FileAST)
			// processed imported packages
			fmt.Printf("FileAST.Imports:\t%#v\n", fileContext.FileAST.Imports)
			if !fileContext.ImportsProcessed {
				missingImports := pp.processImports(fileContext.FileAST.Imports)
				fmt.Printf("Missing:\t\t%#v\n\n", missingImports)
				if len(missingImports) > 0 {
					for _, spec := range missingImports {
						fmt.Printf("Spec:\t\t\t%#v\n", spec)
						c, err := pp.createPackageContext(spec.Path)
						if err != nil {
							return err
						}

						pp.packageStack = append([]*PackageContext{c}, pp.packageStack...)

						fmt.Printf("PackageContext: %#v\n\n", c)
						// byteSlice, _ := json.Marshal(c)
						// fmt.Printf("\nPC: %v\n", string(byteSlice))
					}
					// At least one imported package is not yet processed
					fmt.Printf("----Postponing %v\n\n", p.PackageDir)
					fileContext.ImportsProcessed = true
					continue PACKAGE_STACK
				}
			}
			// All imported packages known => process the AST
			// TODO(jchaloup): reset the ST
			// Keep only the top-most ST
			if err := p.Config.SymbolTable.Reset(); err != nil {
				panic(err)
			}
			payload := fileparser.MakePayload(fileContext.FileAST)
			if err := fileparser.NewParser(p.Config).Parse(payload); err != nil {
				return err
			}
			fmt.Printf("Types: %#v\n", payload.DataTypes)
			fmt.Printf("Vars: %#v\n", payload.Variables)
			fmt.Printf("Funcs: %#v\n", payload.Functions)
			fileContext.DataTypes = payload.DataTypes
			fileContext.Variables = payload.Variables
			fileContext.Functions = payload.Functions
			p.FileIndex++
		}

		// re-process data types
		if err := pp.reprocessDataTypes(p); err != nil {
			return err
		}

		// re-process variables
		if err := pp.reprocessVariables(p); err != nil {
			return err
		}
		// re-process functions
		if err := pp.reprocessFunctions(p); err != nil {
			return err
		}

		// Put the package ST into the global one
		byteSlice, _ := json.Marshal(p.SymbolTable)
		fmt.Printf("\nSymbol table: %v\n\n", string(byteSlice))

		table, err := p.SymbolTable.Table(0)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Global storing %q\n", p.PackagePath)
		if err := pp.globalSymbolTable.Add(p.PackagePath, table); err != nil {
			panic(err)
		}

		// Pop the package from the package stack
		pp.packageStack = pp.packageStack[1:]
	}
	return nil
}

func (pp *ProjectParser) GlobalSymbolTable() *global.Table {
	return pp.globalSymbolTable
}

func printDataType(dataType gotypes.DataType) {
	byteSlice, _ := json.Marshal(dataType)
	fmt.Printf("\n%v\n", string(byteSlice))
}
