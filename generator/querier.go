package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// InterfaceMethod represents a method in an interface
type InterfaceMethod struct {
	Name    string
	Params  []string
	Results []string
}

// findQuerierFiles locates all `_querier.go` files except `querier.go`
func findQuerierFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), "querier_") && info.Name() != "querier.go" {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// parseQuerierMethods extracts all `(q *querier)` methods and import statements
func parseQuerierMethods(filename string) ([]InterfaceMethod, []string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
	if err != nil {
		return nil, nil, err
	}

	var methods []InterfaceMethod
	var imports []string

	// Extract imports
	for _, imp := range node.Imports {
		imports = append(imports, imp.Path.Value) // e.g., `"context"`
	}

	// Extract methods
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Recv != nil {
			if len(fn.Recv.List) == 1 {
				recvType, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
				if ok {
					ident, ok := recvType.X.(*ast.Ident)
					if ok && ident.Name == "querier" {
						method := InterfaceMethod{Name: fn.Name.Name}

						for _, param := range fn.Type.Params.List {
							for _, name := range param.Names {
								method.Params = append(method.Params, fmt.Sprintf("%s %s", name.Name, exprToString(param.Type)))
							}
						}

						if fn.Type.Results != nil {
							for _, result := range fn.Type.Results.List {
								method.Results = append(method.Results, exprToString(result.Type))
							}
						}

						methods = append(methods, method)
					}
				}
			}
		}
		return true
	})

	return methods, imports, nil
}

// exprToString converts an AST expression to its string representation
func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.FuncType:
		return "func(...)"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// generateQuerierFileContent creates the final `querier.go` file content
func generateQuerierFileContent(packageName string, imports []string, methods []InterfaceMethod) string {
	var sb strings.Builder

	// Add auto-generated comment
	sb.WriteString(fmt.Sprintf(`// Code generated. DO NOT EDIT.
// Code generated. DO NOT EDIT.
// Code generated. DO NOT EDIT.

package %v

`, packageName))

	// Add import block
	if len(imports) > 0 {
		sb.WriteString("import (\n")
		importSet := make(map[string]struct{})
		for _, imp := range imports {
			importSet[imp] = struct{}{}
		}
		for imp := range importSet {
			sb.WriteString("\t" + imp + "\n")
		}
		sb.WriteString(")\n\n")
	}

	// Sort methods alphabetically
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Name < methods[j].Name
	})

	// Start interface definition
	sb.WriteString("type Querier interface {\n")
	sb.WriteString("\tWithTx(TxFunc) error\n")
	sb.WriteString("\n")
	sb.WriteString("\t// Auto Generated\n")
	for _, method := range methods {
		sb.WriteString(fmt.Sprintf("\t%s(%s) (%s)\n",
			method.Name,
			strings.Join(method.Params, ", "),
			strings.Join(method.Results, ", "),
		))
	}
	sb.WriteString("}\n\n")

	// Add struct and constructor
	sb.WriteString(`
func NewQuerier(db *gorm.DB, log *zap.Logger) Querier {
	return &querier{db: db, log: log}
}

type querier struct {
	db *gorm.DB
    log *zap.Logger
}

type TxFunc func(Querier) error

func (q *querier) WithTx(fn TxFunc) error {
	return q.db.Transaction(func(tx *gorm.DB) error {
		q := NewQuerier(tx, q.log)
		return fn(q)
	})
}
`)

	return sb.String()
}

// writeQuerierFile replaces `querier.go` with new content
func writeQuerierFile(filename string, content string) error {
	return ioutil.WriteFile(filename, []byte(content), 0644)
}

// postProcessQuerierFile runs `goimports` to fix missing or unused imports
func postProcessQuerierFile(filename string) error {
	cmd := exec.Command("goimports", "-w", filename)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run goimports: %w", err)
	}
	return nil
}

func GenerateQuerier(packageName, dir, querierFileName string) error {
	querierFile := filepath.Join(dir, querierFileName)
	// Step 1: Find all _querier.go files (excluding querier.go)
	files, err := findQuerierFiles(dir)
	if err != nil {
		fmt.Println("Error finding files:", err)
		return err
	}

	var allMethods []InterfaceMethod
	var allImports []string

	for _, file := range files {
		// Step 2: Extract methods and imports
		methods, imports, err := parseQuerierMethods(file)
		if err != nil {
			fmt.Println("Error parsing file:", file, err)
			continue
		}
		allMethods = append(allMethods, methods...)
		allImports = append(allImports, imports...)
	}

	// Step 3: Generate the entire `querier.go` content
	querierContent := generateQuerierFileContent(packageName, allImports, allMethods)

	// Step 4: Write new querier.go file
	err = writeQuerierFile(querierFile, querierContent)
	if err != nil {
		fmt.Println("Error writing querier.go:", err)
		return err
	}

	// Step 5: Run goimports to fix import issues
	err = postProcessQuerierFile(querierFile)
	if err != nil {
		fmt.Println("Warning: goimports failed, you may have unused or missing imports.")
		return err
	}

	fmt.Printf("✅ [%v] Successfully regenerated querier.go with sorted methods and fixed imports\n", packageName)

	return nil
}
