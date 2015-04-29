// Simple Go language function inliner
// Copyright (C) 2015  Steven R. Wiley
// Use of this source code is governed by
// the GNU GENERAL PUBLIC LICENSE
// found in the LICENSE file.
package main

import (
	. "bytes"
	"flag"
	"fmt"
	. "go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
)

type BlockVisitor struct {
	sbytes       Buffer // Holds the source bytes
	sourceCursor int    // Cursor for the source bytes
	pbytes       Buffer // Receives the processed bytes
	// This regexp is used to filter function and variable
	// names for inlining candidates.
	funcNameFilter *regexp.Regexp
	blockOperators []BlockOperator
	inlineFuncs    []*FuncDecl
}

type SubVisitor struct {
	inlines []*AssignStmt // the rhs of the assignment is an inlinable function
	bv      *BlockVisitor
}

type ParamVisitor struct {
	subs             map[string][]byte
	templatePosition int
	bv               *BlockVisitor
}

// BlockOperator functions operate on code blocks and might advance the
// BlockVisitor's sourceCursor variable when called, signifying that the
// block has been written to into the pbytes buffer with whatever
// modifications the operator creates. These are plugins-like functions
// that can be added to expand inliner's abilities. Currently, there are
// three BlockOperators defined; functInline, unwindStaticLoop and assertInline.
type BlockOperator func(st *BlockStmt, m *BlockVisitor)

func (m *BlockVisitor) Visit(n Node) Visitor {
	if n == nil {
		return nil
	}
	switch st := n.(type) {
	case *BlockStmt:
		for _, blockOperator := range m.blockOperators {
			curPos := m.sourceCursor
			blockOperator(st, m)
			if curPos != m.sourceCursor { // block was altered
				return nil // Do not visit children
			}
		}
	}
	return m
}

func (m *ParamVisitor) Visit(n Node) Visitor {
	switch st := n.(type) {
	case *Ident:
		subStr := m.subs[st.Name]
		if subStr != nil {
			if m.templatePosition < int(st.Pos()-1) {
				m.bv.pbytes.Write(m.bv.sbytes.Bytes()[m.templatePosition : st.Pos()-1])
				m.templatePosition = int(st.End() - 1)
			}
			m.bv.pbytes.WriteByte('(')
			m.bv.pbytes.Write(subStr)
			m.bv.pbytes.WriteByte(')')
		}
	}
	return m
}

func (m *SubVisitor) doSubstitution(fNodeType *FuncType, fNodeBody *BlockStmt, tNode *CallExpr) {
	if len(fNodeType.Params.List) != len(tNode.Args) {
		return
	}
	subset := make(map[string][]byte, len(fNodeType.Params.List))
	for i, v := range fNodeType.Params.List {
		s := tNode.Args[i]
		subset[v.Names[0].Name] = m.bv.sbytes.Bytes()[s.Pos()-1 : s.End()-1]
	}
	// Write up to the function call
	if m.bv.sourceCursor < int(tNode.Pos())-1 {
		m.bv.pbytes.Write(m.bv.sbytes.Bytes()[m.bv.sourceCursor : tNode.Pos()-1])
		m.bv.sourceCursor = int(tNode.Pos()) - 1
	}
	m.bv.sourceCursor = int(tNode.End()) - 1 // Skips the rest of target tNode
	i := int(fNodeBody.Pos())
loop: // Advance past the first \n and \t's of the inline function; irrelevant if the
	//  generator calls go fmt
	for ; ; i++ {
		switch m.bv.sbytes.Bytes()[i] {
		case '\t', '\n':
		default:
			break loop
		}
	}
	pv := &ParamVisitor{subset, i, m.bv}
	Walk(pv, fNodeBody)
	if pv.templatePosition < int(fNodeBody.End())-1 {
		m.bv.pbytes.Write(TrimRight(
			m.bv.sbytes.Bytes()[pv.templatePosition:int(fNodeBody.End())-2], "\n\t"))
		m.bv.pbytes.WriteString(" // inlined ")
		m.bv.pbytes.Write(m.bv.sbytes.Bytes()[tNode.Pos()-1 : tNode.End()-1])
	}
}

func (m *SubVisitor) Visit(n Node) Visitor {
	switch st := n.(type) {
	case *CallExpr:
		tfnc, ok := st.Fun.(*Ident)
		if !ok {
			break
		}
		for _, assign := range m.inlines {
			lh, _ := assign.Lhs[0].(*Ident)
			if lh.Name == tfnc.Name {
				infunc, ok := assign.Rhs[0].(*FuncLit)
				if !ok { // This should have been pre-checked and never fire
					continue
				}
				m.doSubstitution(infunc.Type, infunc.Body, st)
			}
		}
		for _, funcDecl := range m.bv.inlineFuncs {
			if funcDecl.Name.Name == tfnc.Name {
				m.doSubstitution(funcDecl.Type, funcDecl.Body, st)
			}
		}
	}
	return m
}

func isInlineable(sm *AssignStmt, fileFilter *regexp.Regexp) (yes bool) {
	if len(sm.Lhs) != 1 { // only single assignments allowed
		return
	}
	if sm.Tok != token.DEFINE && sm.Tok != token.ASSIGN {
		return
	}
	lh, ok := sm.Lhs[0].(*Ident)
	if !ok || len(fileFilter.FindString(lh.Name)) == 0 {
		return
	}
	fLit, ok := sm.Rhs[0].(*FuncLit)
	if !ok {
		return
	}
	if fLit.Type.Results != nil {
		return
	}
	return true
}

// Test if the loop has an integer loop counter variable with
// static bounds and simple incrementer
func isUnwindable(f *ForStmt, fileFilter *regexp.Regexp) (
	canUnwind bool, startVal, endVal int, identName string) {
	assign, ok := f.Init.(*AssignStmt)
	if !ok {
		return
	}
	// Test for single assignment from interger literal
	if len(assign.Lhs) != 1 || len(assign.Lhs) != 1 || assign.Tok != token.DEFINE {
		return
	}
	ident, ok := assign.Lhs[0].(*Ident)
	if !ok || len(fileFilter.FindString(ident.Name)) == 0 {
		return
	}
	identName = ident.Name
	bLit, ok := assign.Rhs[0].(*BasicLit)
	if !ok || bLit.Kind != token.INT {
		return
	}
	var err error
	startVal, err = strconv.Atoi(bLit.Value)
	if err != nil {
		return
	}
	// Test for simple conditional
	binExrp, ok := f.Cond.(*BinaryExpr)
	if !ok {
		return
	}
	ident2, ok := binExrp.X.(*Ident)
	if !ok || ident2.Name != ident.Name {
		return
	}
	blit2, ok := binExrp.Y.(*BasicLit)
	if !ok || blit2.Kind != token.INT {
		return
	}
	endVal, err = strconv.Atoi(blit2.Value)
	if err != nil {
		return
	}
	if binExrp.Op == token.LEQ {
		endVal++
	}
	// Test incrementer
	inds, ok := f.Post.(*IncDecStmt)
	if !ok {
		return
	}
	ident3, ok := binExrp.X.(*Ident)
	if !ok || ident3.Name != ident.Name || inds.Tok != token.INC {
		return
	}
	canUnwind = true
	return
}

// Tests if an ExperStmt has an affirm or deny string
func canInlineAssert(sm *ExprStmt) (yes bool, callexpr *CallExpr, action string, name string) {
	callexpr, ok := sm.X.(*CallExpr)
	if !ok {
		return
	}
	tfnc, ok := callexpr.Fun.(*Ident)
	if !ok {
		return
	}
	name = tfnc.Name
	if name != "affirm_" && name != "deny_" {
		return
	}
	action = "return"
	if len(callexpr.Args) == 2 {
		switch bl := callexpr.Args[1].(type) {
		case *BasicLit:
			if bl.Kind != token.STRING {
				return
			}
			action = string(Trim([]byte(bl.Value), "\"`"))
			yes = true
			return
		default:
			return
		}
	}
	if len(callexpr.Args) != 1 { // There needs to be at least one argument
		return
	}
	yes = true
	return
}

// Inlines asserts with the keywords 'affirm_' or 'deny_'.
var assertInline BlockOperator = func(f *BlockStmt, m *BlockVisitor) {
blockList:
	for _, statement := range f.List {
		switch sm := statement.(type) {
		case *ExprStmt:
			yes, callexpr, action, name := canInlineAssert(sm)
			if !yes {
				continue blockList
			}
			pos := (name == "affirm_")
			// Write up to the for loop to unwind
			if m.sourceCursor < int(sm.Pos())-1 {
				m.pbytes.Write(m.sbytes.Bytes()[m.sourceCursor : sm.Pos()-1])
			}
			// Comment out the source assertion
			m.pbytes.WriteString("/* ")
			m.pbytes.Write(m.sbytes.Bytes()[sm.Pos()-1 : sm.End()-1])
			m.pbytes.WriteString(" /* inlined assert */\n")

			// Write the assertion
			m.pbytes.WriteString("if ")
			switch callexpr.Args[0].(type) {
			case *BinaryExpr:
				if pos {
					m.pbytes.WriteString("(")
				}
				m.pbytes.Write(m.sbytes.Bytes()[callexpr.Args[0].Pos()-1 : callexpr.Args[0].End()-1])
				if pos {
					m.pbytes.WriteString(")")
				}
				var opStr = ""
				if pos {
					opStr = "== false "
				}
				m.pbytes.WriteString(opStr + " { " + action + " } /* */")
			case *Ident:
				m.pbytes.Write(m.sbytes.Bytes()[callexpr.Args[0].Pos()-1 : callexpr.Args[0].End()-1])
				var opStr = "=="
				if !pos {
					opStr = "!="
				}
				m.pbytes.WriteString(" " + opStr + " nil { " + action + " } /* */ ")
			}
			// Advance the source cursor to the end of the assert statement
			m.sourceCursor = int(sm.End()) - 1
		}
	}
}

// Unwinds for statements conforming to strict static loop requirements
var unwindStaticLoop BlockOperator = func(f *BlockStmt, m *BlockVisitor) {
	for _, statement := range f.List {
		switch sm := statement.(type) {
		case *ForStmt:
			canUnwind, startVal, endVal, identName := isUnwindable(sm, m.funcNameFilter)
			if canUnwind {
				subset := make(map[string][]byte, 1)
				// Write up to the for loop to unwind
				if m.sourceCursor < int(sm.Pos())-1 {
					m.pbytes.Write(m.sbytes.Bytes()[m.sourceCursor : sm.Pos()-1])
				}
				// comment the for loop
				m.pbytes.WriteString(" /* ")
				m.pbytes.Write(m.sbytes.Bytes()[sm.Pos()-1 : sm.Body.Lbrace])
				m.pbytes.WriteString(" /* unwound */ ")
				for i := startVal; i < endVal; i++ {
					subset[identName] = []byte(strconv.Itoa(i))
					t := int(sm.Body.Pos())
					pv := &ParamVisitor{subset, t, m}
					Walk(pv, sm.Body)
					if pv.templatePosition < int(sm.Body.End())-1 {
						m.pbytes.Write(TrimRight(
							m.sbytes.Bytes()[pv.templatePosition:int(sm.Body.End())-2],
							"\n\t"))
					}
				}
				m.pbytes.WriteString(" /* } */ ")
				// Advance the source cursor to the end of the for loop
				m.sourceCursor = int(sm.Body.End()) - 1
			}
		}
	}
}

var functInline BlockOperator = func(f *BlockStmt, m *BlockVisitor) {
	var inlines []*AssignStmt
	for _, statement := range f.List {
		switch sm := statement.(type) {
		case *AssignStmt:
			if isInlineable(sm, m.funcNameFilter) {
				inlines = append(inlines, sm)
			}
		}
	}
	if len(inlines) > 0 {
		curPlace := m.sourceCursor
		Walk(&SubVisitor{inlines: inlines, bv: m}, f)
		if curPlace == m.sourceCursor { // The final cycle can be used to comment out
			// the template function since the m.sourceCursor has not been advanced
			for _, infnc := range inlines {
				m.pbytes.Write(m.sbytes.Bytes()[m.sourceCursor : infnc.Pos()-1])
				m.pbytes.WriteString(" /* ")
				m.pbytes.Write(m.sbytes.Bytes()[infnc.Pos()-1 : infnc.End()-1])
				m.pbytes.WriteString(" /* inlined func */ ")
				m.sourceCursor = int(infnc.End()) - 1
			}
		}
	}
	return
}

func (bv *BlockVisitor) collectTopLevelCandidates(f *File) {
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *FuncDecl:
			if len(bv.funcNameFilter.FindString(d.Name.Name)) == 0 {
				break
			}
			// Only functions without receivers allowed
			if d.Recv != nil {
				return
			}
			// Only functions without results allowed
			if d.Type.Results != nil {
				return
			}
			bv.inlineFuncs = append(bv.inlineFuncs, d)
		}
	}
}

func Inline(firstBytes []byte, out io.Writer, fileFilterRegex string) (rErr error) {
	fileFilter, err := regexp.Compile(fileFilterRegex)
	if err != nil {
		return err
	}

	// Trim the '+build generate' directive from the file if present
	importDecl := regexp.MustCompile(`(^|[\n])\/\/\s+\+build\s+generate\s?[\n]`)
	imIndex := importDecl.FindIndex(firstBytes)
	if imIndex != nil {
		firstBytes = firstBytes[imIndex[1]:]
	}
	// Load up a slice of BlockOperator types with the three available block Operators
	ops := []BlockOperator{functInline, unwindStaticLoop, assertInline}
	bv := &BlockVisitor{sbytes: *NewBuffer(firstBytes), blockOperators: ops,
		funcNameFilter: fileFilter}
	cycles := 0
	for fired := true; fired; {
		cycles++
		fset := token.NewFileSet() // positions are relative to fset apparently ?
		myAst, err := parser.ParseFile(fset, "", bv.sbytes.Bytes(), parser.AllErrors)
		if err != nil {
			return err
		}
		if cycles == 1 {
			bv.collectTopLevelCandidates(myAst)
		}
		Walk(bv, myAst)
		//Print(fset, myAst)
		//os.Exit(0)
		if bv.sourceCursor < len(bv.sbytes.Bytes()) {
			bv.pbytes.Write(bv.sbytes.Bytes()[bv.sourceCursor:])
		}
		fired = bv.sourceCursor != 0 // Was anything done?
		if fired {
			bv.sbytes, bv.pbytes = bv.pbytes, bv.sbytes // Swap source and processed byte bufers
			bv.pbytes.Reset()                           // Clear the processed pad
			bv.sourceCursor = 0
		}
	}
	out.Write(bv.sbytes.Bytes())
	return
}

func InlineFile(fileName string, w io.Writer, fileFilter string) (rErr error) {
	firstBytes, rErr := ioutil.ReadFile(fileName)
	if rErr != nil {
		return
	}
	_, rErr = w.Write([]byte("// This file is generated by inliner. DO NOT EDIT.\n"))
	if rErr != nil {
		return
	}
	_, rErr = w.Write([]byte("// Source file: " + fileName + "\n"))
	if rErr != nil {
		return
	}
	return Inline(firstBytes, w, fileFilter)
}

func main() {
	var outputFile, inputFile, fileFilter string
	help := false
	flag.BoolVar(&help, "help", false, "Print arguments")
	flag.StringVar(&outputFile, "out", "", "Name of output file")
	flag.StringVar(&inputFile, "in", "", "Name of input file")
	flag.StringVar(&fileFilter, "filter", "_$", "Regular expression to filter inlineable names.")
	flag.Parse()
	if len(outputFile) == 0 || len(inputFile) == 0 {
		fmt.Println("Illegal command arguments")
		flag.CommandLine.PrintDefaults()
		return
	}
	if help {
		fmt.Println("inliner utility for Go language intended for use with go generate")
		flag.CommandLine.PrintDefaults()
		return
	}
	w, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "File ", outputFile, " Outfile create error ", err)
		return
	}
	defer w.Close()
	err = InlineFile(inputFile, w, fileFilter)
	if err != nil {
		fmt.Fprintln(os.Stderr, "inline error", err)
	}
}
