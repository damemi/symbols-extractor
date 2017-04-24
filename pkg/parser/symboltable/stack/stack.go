package stack

import (
	"fmt"

	"github.com/gofed/symbols-extractor/pkg/parser/symboltable"
	gotypes "github.com/gofed/symbols-extractor/pkg/types"
)

// Stack is a multi-level symbol table for parsing blocks of code
type Stack struct {
	Tables []*symboltable.Table `json:"tables"`
	Size   int                  `json:"size"`
}

// NewStack creates an empty stack with no symbol table
func New() *Stack {
	return &Stack{
		Tables: make([]*symboltable.Table, 0),
		Size:   0,
	}
}

// Push pushes a new symbol table at the top of the stack
func (s *Stack) Push() {
	s.Tables = append(s.Tables, symboltable.NewTable())
	s.Size++
	fmt.Printf("Push: %v\n", s.Size)
}

// Pop pops the top most symbol table from the stack
func (s *Stack) Pop() {
	if s.Size > 0 {
		s.Tables = s.Tables[:s.Size-1]
		s.Size--
	} else {
		panic("Popping over an empty stack of symbol tables")
		// If you reached this line you are a magician
	}
}

func (s *Stack) AddVariable(sym *gotypes.SymbolDef) error {
	if s.Size > 0 {
		fmt.Printf("====Adding %v variable at level %v\n", sym.Name, s.Size-1)
		return s.Tables[s.Size-1].AddVariable(sym)
	}
	return fmt.Errorf("Symbol table stack is empty")
}

func (s *Stack) AddDataType(sym *gotypes.SymbolDef) error {
	if s.Size > 0 {
		fmt.Printf("====Adding %#v datatype at level %v\n", sym, s.Size-1)
		return s.Tables[s.Size-1].AddDataType(sym)
	}
	return fmt.Errorf("Symbol table stack is empty")
}

func (s *Stack) AddFunction(sym *gotypes.SymbolDef) error {
	if s.Size > 0 {
		return s.Tables[s.Size-1].AddFunction(sym)
	}
	return fmt.Errorf("Symbol table stack is empty")
}

func (s *Stack) LookupVariable(name string) (*gotypes.SymbolDef, error) {
	// The top most item on the stack is the right most item in the simpleSlice
	for i := s.Size - 1; i >= 0; i-- {
		def, err := s.Tables[i].LookupVariable(name)
		if err == nil {
			fmt.Printf("Table %v: symbol: %#v\n", i, def)
			return def, nil
		}
	}
	return nil, fmt.Errorf("Symbol %v not found", name)
}

// Lookup looks for the first occurrence of a symbol with the given name
func (s *Stack) Lookup(name string) (*gotypes.SymbolDef, symboltable.SymbolType, error) {
	// The top most item on the stack is the right most item in the simpleSlice
	for i := s.Size - 1; i >= 0; i-- {
		def, st, err := s.Tables[i].Lookup(name)
		if err == nil {
			fmt.Printf("Table %v: symbol: %#v\n", i, def)
			return def, st, nil
		}
	}
	return nil, symboltable.SymbolType(""), fmt.Errorf("Symbol %v not found", name)
}

func (s *Stack) Reset(level int) error {
	fmt.Printf("level: %v, size: %v, a: %v, b: %v\n", level, s.Size, level < 0, (s.Size-1) < level)
	if level < 0 || (s.Size-1) < level {
		return fmt.Errorf("Level %v out of range", level)
	}
	if level == 0 {
		s.Tables = s.Tables[:1]
		s.Size = 1
	} else {
		s.Tables = s.Tables[:level+1]
		s.Size = level + 1
	}
	return nil
}

// Table gets a symbol table at given level
// Level 0 corresponds to the file level symbol table (the top most block)
func (s *Stack) Table(level int) (*symboltable.Table, error) {
	if level < 0 || s.Size-1 < level {
		return nil, fmt.Errorf("No symbol table found for level %v", level)
	}
	return s.Tables[level], nil
}

func (s *Stack) Print() {
	for i := s.Size - 1; i >= 0; i-- {
		fmt.Printf("Table %v: symbol: %#v\n", i, s.Tables[i])
	}
}

func (s *Stack) PrintTop() {
	fmt.Printf("TableSymbols: %#v\n", s.Tables[s.Size-1])
}
