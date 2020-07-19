package fxparse

import (
	"fmt"
	"fxsym"
	"strings"
)

const nullString = "nil"

/*type TreeSym fxsym.Sym

func (s *TreeSym) Content() interface{} {
	return s.Content()
}

func (s *TreeSym) GetFunc() (f *Func, err error) {
	if f, ok := s.Content().(*Func); ok {
		return f, nil
	} else {
		return nil, errors.New("cast to Func failed")
	}
}*/

type Prog struct {
	funcs []*fxsym.Sym
	depth int
}

func NewProg() (prog *Prog) {
	prog = &Prog{depth: 0}
	prog.funcs = nil

	return prog
}

func (p *Prog) AddFunc(f *fxsym.Sym) {
	if f != nil {
		p.funcs = append(p.funcs, f)
	}
}

func (p *Prog) String() string {
	if p == nil {
		return nullString
	}

	tabs := strings.Repeat("\t", p.depth)
	output := fmt.Sprintf("%s%p PROG", tabs, p)
	for _, value := range p.funcs {
		value.SetDepth(p.depth + 1)
		output += fmt.Sprintf("\n%s", value)
		value.Content().(*Func).depth = p.depth + 2
		output += fmt.Sprintf("\n%s", value.Content())
	}

	return output
}

type Func struct {
	head  *Head
	body  *Body
	depth int
}

func NewFunc() (f *Func) {
	f = &Func{depth: 0}
	f.head = NewHead()
	f.body = NewBody()

	return f
}

func (f *Func) String() string {
	if f == nil {
		return nullString
	}

	tabs := strings.Repeat("\t", f.depth)
	output := fmt.Sprintf("%s%p FUNC\n", tabs, f)
	// Head
	f.head.depth = f.depth + 1
	output += fmt.Sprintf("%s\n", f.head)
	// Body
	f.body.depth = f.depth + 1
	output += fmt.Sprintf("%s", f.body)

	return output
}

type Head struct {
	id     string
	params []*fxsym.Sym
	depth  int
}

func NewHead() (head *Head) {
	head = &Head{depth: 0}

	return head
}

func (h *Head) AddParam(param *fxsym.Sym) {
	if param != nil {
		h.params = append(h.params, param)
	}
}

func (h *Head) String() string {
	if h == nil {
		return nullString
	}

	tabs := strings.Repeat("\t", h.depth)
	output := fmt.Sprintf("%s%p HEAD(%s)", tabs, h, h.id)
	for _, value := range h.params {
		value.SetDepth(h.depth + 1)
		output += fmt.Sprintf("\n%s", value)
	}

	return output
}

type Body struct {
	stms  []*Statement
	depth int
}

func NewBody() (body *Body) {
	body = &Body{depth: 0}
	body.stms = nil

	return body
}

func (b *Body) AddStm(stm *Statement) {
	if stm != nil {
		b.stms = append(b.stms, stm)
	}
}

func (b *Body) String() string {
	if b == nil {
		return nullString
	}

	tabs := strings.Repeat("\t", b.depth)
	output := fmt.Sprintf("%s%p BODY", tabs, b)
	for _, value := range b.stms {
		value.depth = b.depth + 1
		output += fmt.Sprintf("\n%s", value)
	}

	return output
}

type Statement struct {
	// One of these
	call  *Call
	iter  *Iter
	body  *Body
	decl  *fxsym.Sym
	depth int
}

func NewStatement() (stm *Statement) {
	stm = &Statement{depth: 0}
	stm.call = nil
	stm.iter = nil
	stm.body = nil
	stm.decl = nil

	return stm
}

func (stm *Statement) AddCall(call *Call) {
	if call != nil {
		stm.call = call
	}
}

func (stm *Statement) AddIter(iter *Iter) {
	if iter != nil {
		stm.iter = iter
	}
}

func (stm *Statement) AddBody(body *Body) {
	if body != nil {
		stm.body = body
	}
}

func (stm *Statement) AddDecl(decl *fxsym.Sym) {
	if decl != nil {
		stm.decl = decl
	}
}

func (stm *Statement) String() string {
	if stm == nil {
		return nullString
	}

	if stm.call != nil {
		stm.call.depth = stm.depth
		return fmt.Sprintf("%s", stm.call)
	} else if stm.iter != nil {
		stm.iter.depth = stm.depth
		return fmt.Sprintf("%s", stm.iter)
	} else if stm.body != nil {
		stm.body.depth = stm.depth
		return fmt.Sprintf("%s", stm.body)
	} else if stm.decl != nil {
		stm.decl.SetDepth(stm.depth)
		return fmt.Sprintf("%s", stm.decl)
	}

	return nullString
}

type Call struct {
	f     *fxsym.Sym
	args  []*Expr
	depth int
}

func NewCall() (call *Call) {
	call = &Call{depth: 0}
	call.f = nil
	call.args = nil

	return call
}

func (c *Call) AddFunc(f *fxsym.Sym) {
	if f != nil {
		c.f = f
	}
}

func (c *Call) AddArg(arg *Expr) {
	if arg != nil {
		c.args = append(c.args, arg)
	}
}

func (c *Call) String() string {
	if c == nil {
		return nullString
	}

	tabs := strings.Repeat("\t", c.depth)
	output := fmt.Sprintf("%s%p CALL", tabs, c)
	// Func
	c.f.SetDepth(c.depth + 1)
	output += fmt.Sprintf("\n%s", c.f)
	// Args
	for _, value := range c.args {
		value.depth = c.depth + 1
		output += fmt.Sprintf("\n%s", value)
	}

	return output
}

type Iter struct {
	varControl *fxsym.Sym
	start      *Expr
	end        *Expr
	step       *Expr
	body       *Body
	depth      int
}

func NewIter() (iter *Iter) {
	iter = &Iter{depth: 0}
	iter.body = NewBody()

	return iter
}

func (iter *Iter) AddVarControl(v *fxsym.Sym) {
	if v != nil {
		iter.varControl = v
	}
}

func (iter *Iter) AddStart(e *Expr) {
	if e != nil {
		iter.start = e
	}
}

func (iter *Iter) AddEnd(e *Expr) {
	if e != nil {
		iter.end = e
	}
}

func (iter *Iter) AddStep(e *Expr) {
	if e != nil {
		iter.step = e
	}
}

func (iter *Iter) AddBody(b *Body) {
	if b != nil {
		iter.body = b
	}
}

func (iter *Iter) String() string {
	if iter == nil {
		return nullString
	}

	tabs := strings.Repeat("\t", iter.depth)
	output := fmt.Sprintf("%s%p ITER\n", tabs, iter)
	// Control variable
	iter.varControl.SetDepth(iter.depth + 1)
	output += fmt.Sprintf("%s\n", iter.varControl)
	// Start
	iter.start.depth = iter.depth + 1
	output += fmt.Sprintf("%s\n", iter.start)
	// End
	iter.end.depth = iter.depth + 1
	output += fmt.Sprintf("%s\n", iter.end)
	// Step
	iter.step.depth = iter.depth + 1
	output += fmt.Sprintf("%s\n", iter.step)
	// Body
	iter.body.depth = iter.depth + 1
	output += fmt.Sprintf("%s", iter.body)

	return output
}

type Expr struct {
	atom  *fxsym.Sym
	depth int
}

func NewExpr(sTok *fxsym.Sym) (expr *Expr) {
	expr = &Expr{depth: 0}
	if sTok != nil {
		expr.atom = sTok
	}

	return expr
}

func (e *Expr) String() string {
	if e == nil {
		return nullString
	}

	tabs := strings.Repeat("\t", e.depth)
	output := fmt.Sprintf("%s%p EXPR", tabs, e)
	// Atom
	e.atom.SetDepth(e.depth + 1)
	output += fmt.Sprintf("\n%s", e.atom)

	return output
}
