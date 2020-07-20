package fxparse

import (
	"fmt"
	"fxlex"
	"fxsym"
	"os"
	"strings"
)

const maxErrors = 5

var DebugParser bool = false

type Parser struct {
	l      *fxlex.Lexer
	nErr   int
	depth  int
	stkEnv fxsym.StkEnv
}

func NewParser(l *fxlex.Lexer) (p *Parser, err error) {
	p = &Parser{l, 0, 0, nil}

	p.stkEnv.PushEnv()
	p.initSyms()
	p.stkEnv.PushEnv()

	return p, nil
}

func (p *Parser) Parse() error {
	p.pushTrace("Parse")
	defer p.popTrace()

	prog := NewProg()
	if err := p.Prog(prog); err != nil {
		return err
	}

	if p.nErr == 0 {
		fmt.Println(prog)
	}

	return nil
}

func (p *Parser) initSyms() error {
	p.defBuiltins()
	p.defTypes()
	return nil
}

func (p *Parser) defBuiltins() error {
	for name, builtin := range builtins {
		f := NewFunc()
		f.head.id = builtin.name

		p.stkEnv.PushEnv()
		for i, arg := range builtin.args {
			vSym, err := p.stkEnv.NewSym(arg, fxsym.SVar)
			if err != nil {
				return err
			}
			vSym.AddTokKind(fxlex.TokKey)
			vSym.AddPlace("builtin", i)
			f.head.AddParam(vSym)
		}
		p.stkEnv.PopEnv()

		fSym, err := p.stkEnv.NewSym(name, fxsym.SFunc)
		if err != nil {
			return err
		}
		fSym.AddTokKind(builtin.kind)
		fSym.AddPlace("builtin", 0)
		fSym.AddContent(f)
	}

	return nil
}

func (p *Parser) defTypes() error {
	for _, tp := range Types {
		tSym, err := p.stkEnv.NewSym(tp.String(), fxsym.SType)
		if err != nil {
			return err
		}
		tSym.AddTokKind(fxlex.TokID)
		tSym.AddPlace("builtin", 0)
		tSym.AddContent(tp)
	}

	return nil
}

func (p *Parser) match(tT int) (t fxlex.Token, isMatch bool, e error) {
	t, err := p.l.Peek()
	if err != nil {
		return fxlex.Token{}, false, err
	}

	if t.GetTokType() != tT {
		if t.GetTokType() == fxlex.TokEOF {
			panic("unexpected EOF")
		}

		return t, false, nil
	}

	t, _ = p.l.Lex()
	return t, true, nil
}

func (p *Parser) errorf(s string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, s+"\n", v...)
	p.nErr++
	if p.nErr >= maxErrors {
		panic("too many errors")
	}
}

func (p *Parser) pushTrace(tag string) {
	if DebugParser {
		tabs := strings.Repeat("\t", p.depth)
		fmt.Fprintf(os.Stderr, "%s%s\n", tabs, tag)
	}
	p.depth++
}

func (p *Parser) popTrace() {
	p.depth--
}

// <PROG> ::= 'func' <FUNC> <PROG> |
//            'EOF'
func (p *Parser) Prog(prog *Prog) error {
	p.pushTrace("Prog")
	defer p.popTrace()

	t, err := p.l.Peek()
	if err != nil {
		return err
	}

	switch t.GetTokType() {
	case fxlex.TokFunc:
		t, err = p.l.Lex()
		p.pushTrace("\"func\"")
		p.popTrace()

		f, err := p.Func()
		if err != nil {
			return err
		}

		fSym, err := p.stkEnv.NewSym(f.head.id, fxsym.SFunc)
		if err != nil {
			p.errorf("%s:%d: syntax error: %s (%s)",
				p.l.GetFilename(), p.l.GetLineNumber(), err, f.head.id)
		} else {
			fSym.AddTokKind(fxlex.TokFunc)
			fSym.AddPlace(p.l.GetFilename(), p.l.GetLineNumber())
			fSym.AddContent(f)
		}

		prog.AddFunc(fSym)

		return p.Prog(prog)
	case fxlex.TokEOF:
		t, err = p.l.Lex()
		p.pushTrace("\"EOF\"")
		p.popTrace()
	default:
		p.errorf("%s:%d: syntax error: expected func or EOF, found %s",
			p.l.GetFilename(), p.l.GetLineNumber(), t.GetTokType())
	}

	return err
}

// <FUNC> ::= <HEAD> '{' <BODY> '}'
func (p *Parser) Func() (f *Func, err error) {
	p.pushTrace("Func")
	defer p.popTrace()

	p.stkEnv.PushEnv()
	defer p.stkEnv.PopEnv()

	f = NewFunc()

	if err := p.Head(f.head); err != nil {
		return nil, err
	}

	t, isLCurl, err := p.match(fxlex.TokLCurl)
	if err != nil {
		return nil, err
	} else if !isLCurl {
		p.errorf("%s:%d: syntax error: macro bad definition",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokLCurl, fxlex.TokRCurl)
		if err != nil {
			return nil, err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	if err := p.Body(f.body); err != nil {
		return nil, err
	}

	t, isRCurl, err := p.match(fxlex.TokRCurl)
	if err != nil {
		return nil, err
	} else if !isRCurl {
		p.errorf("%s:%d: syntax error: macro bad definition",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntilAndLex(fxlex.TokRCurl)
		return nil, err
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	return f, err
}

// <HEAD> ::= id '(' <FORMAL_PRMS> ')'
func (p *Parser) Head(head *Head) error {
	p.pushTrace("Head")
	defer p.popTrace()

	t, isID, err := p.match(fxlex.TokID)
	if err != nil {
		return err
	} else if !isID {
		p.errorf("%s:%d: syntax error: macro bad definition",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokLPar, fxlex.TokRPar, fxlex.TokComma)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("ID %s", t))
		p.popTrace()

		head.id = t.GetLexeme()
	}

	t, isLPar, err := p.match(fxlex.TokLPar)
	if err != nil {
		return err
	} else if !isLPar {
		p.errorf("%s:%d: syntax error: macro bad definition",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokRPar, fxlex.TokComma)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	if err := p.FormalPrms(head); err != nil {
		return err
	}

	t, isRPar, err := p.match(fxlex.TokRPar)
	if err != nil {
		return err
	} else if !isRPar {
		p.errorf("%s:%d: syntax error: macro bad definition",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntilAndLex(fxlex.TokRPar)
		return err
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	return err
}

// <FORMAL_PRMS> ::= type_id id <PRMS> |
//                   <Empty>
func (p *Parser) FormalPrms(head *Head) error {
	p.pushTrace("FormalPrms")
	defer p.popTrace()

	tokType, isTypeID, err := p.match(fxlex.TokID)
	if err != nil || !isTypeID {
		return err
	}

	p.pushTrace(fmt.Sprintf("TypeID %s", tokType))
	p.popTrace()

	tSym := p.stkEnv.GetSym(tokType.GetLexeme())
	if tSym == nil {
		p.errorf("%s:%d: syntax error: type %s not found",
			p.l.GetFilename(), p.l.GetLineNumber(), tokType.GetLexeme())
	} else if tSym.SymType() != "SType" {
		p.errorf("%s:%d: syntax error: expecting type, found %s",
			p.l.GetFilename(), p.l.GetLineNumber(), tokType.GetLexeme())
		tSym = nil
	}

	tokID, isID, err := p.match(fxlex.TokID)
	if err != nil {
		return err
	} else if !isID {
		p.errorf("%s:%d: syntax error: macro bad definition",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokRPar, fxlex.TokComma)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("ID %s", tokID))
		p.popTrace()
	}

	vSym, err := p.stkEnv.NewSym(tokID.GetLexeme(), fxsym.SVar)
	if err != nil {
		p.errorf("%s:%d: syntax error: %s (%s)",
			p.l.GetFilename(), p.l.GetLineNumber(), err, tokID.GetLexeme())
	} else {
		vSym.AddTokKind(tokType.GetTokType())
		vSym.AddPlace(p.l.GetFilename(), p.l.GetLineNumber())
		if tSym != nil {
			vSym.SetType(tSym.Content().(*Type).id)
		}
	}

	head.AddParam(vSym)

	return p.Prms(head)
}

// <PRMS> ::= ',' type_id id <PRMS>
//            <Empty>
func (p *Parser) Prms(head *Head) error {
	p.pushTrace("Prms")
	defer p.popTrace()

	t, isComma, err := p.match(fxlex.TokComma)
	if err != nil || !isComma {
		return err
	}

	p.pushTrace(fmt.Sprintf("%s", t))
	p.popTrace()

	tokType, isTypeID, err := p.match(fxlex.TokID)
	if err != nil {
		return err
	} else if !isTypeID {
		p.errorf("%s:%d: syntax error: macro bad definition",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokRPar, fxlex.TokComma)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("TypeID %s", tokType))
		p.popTrace()
	}

	tSym := p.stkEnv.GetSym(tokType.GetLexeme())
	if tSym == nil {
		p.errorf("%s:%d: syntax error: type %s not found",
			p.l.GetFilename(), p.l.GetLineNumber(), tokType.GetLexeme())
	} else if tSym.SymType() != "SType" {
		p.errorf("%s:%d: syntax error: expecting type, found %s",
			p.l.GetFilename(), p.l.GetLineNumber(), tokType.GetLexeme())
		tSym = nil
	}

	tokID, isID, err := p.match(fxlex.TokID)
	if err != nil {
		return err
	} else if !isID {
		p.errorf("%s:%d: syntax error: macro bad definition",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokRPar, fxlex.TokComma)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("ID %s", tokID))
		p.popTrace()
	}

	vSym, err := p.stkEnv.NewSym(tokID.GetLexeme(), fxsym.SVar)
	if err != nil {
		p.errorf("%s:%d: syntax error: %s (%s)",
			p.l.GetFilename(), p.l.GetLineNumber(), err, tokID.GetLexeme())
	} else {
		vSym.AddTokKind(tokType.GetTokType())
		vSym.AddPlace(p.l.GetFilename(), p.l.GetLineNumber())
		if tSym != nil {
			vSym.SetType(tSym.Content().(*Type).id)
		}
	}

	head.AddParam(vSym)

	return p.Prms(head)
}

// <BODY> ::= id '(' <CALL> <BODY> |
//            'iter' <ITER> <BODY> |
//            type_id id ; <BODY> |
//            var_id '=' <EXPR> ';' <BODY> |
//            '{' <BODY> '}' |
//            <Empty>
func (p *Parser) Body(body *Body) error {
	p.pushTrace("Body")
	defer p.popTrace()

	p.stkEnv.PushEnv()
	defer p.stkEnv.PopEnv()

	t, err := p.l.Peek()
	if err != nil {
		return err
	}

	stm := NewStatement()

	switch t.GetTokType() {
	case fxlex.TokID:
		tokID, _ := p.l.Lex()
		p.pushTrace(fmt.Sprintf("ID %s", tokID))
		p.popTrace()

		sym := p.stkEnv.GetSym(tokID.GetLexeme())
		if sym == nil {
			p.errorf("%s:%d: syntax error: symbol %s not found",
				p.l.GetFilename(), p.l.GetLineNumber(), tokID.GetLexeme())
		}

		switch sym.SymType() {
		case "SFunc":
			call := NewCall()
			call.AddFunc(sym)

			t, isLPar, err := p.match(fxlex.TokLPar)
			if err != nil {
				return err
			} else if !isLPar {
				p.errorf("%s:%d: syntax error: bad statement",
					p.l.GetFilename(), p.l.GetLineNumber())
				err = p.l.SkipUntil(fxlex.TokRPar, fxlex.TokComma, fxlex.Semicolon)
				if err != nil {
					return err
				}
			} else {
				p.pushTrace(fmt.Sprintf("%s", t))
				p.popTrace()
			}

			if err := p.Call(call); err != nil {
				return err
			}

			stm.AddCall(call)
		case "SType":
			tokID, isID, err := p.match(fxlex.TokID)
			if err != nil {
				return err
			} else if !isID {
				p.errorf("%s:%d: syntax error: bad statement",
					p.l.GetFilename(), p.l.GetLineNumber())
				err = p.l.SkipUntil(fxlex.Semicolon)
				if err != nil {
					return err
				}
			} else {
				p.pushTrace(fmt.Sprintf("ID %s", tokID))
				p.popTrace()
			}

			vSym, err := p.stkEnv.NewSym(tokID.GetLexeme(), fxsym.SVar)
			if err != nil {
				p.errorf("%s:%d: syntax error: %s (%s)",
					p.l.GetFilename(), p.l.GetLineNumber(), err, tokID.GetLexeme())
			} else {
				if sym != nil {
					vSym.SetType(sym.Content().(*Type).id)
				}
				vSym.AddPlace(p.l.GetFilename(), p.l.GetLineNumber())
			}

			t, isSemicolon, err := p.match(fxlex.Semicolon)
			if err != nil {
				return err
			} else if !isSemicolon {
				p.errorf("%s:%d: syntax error: bad statement",
					p.l.GetFilename(), p.l.GetLineNumber())
				err = p.l.SkipUntilAndLex(fxlex.Semicolon)
				return err
			} else {
				p.pushTrace(fmt.Sprintf("%s", t))
				p.popTrace()
			}

			stm.AddDecl(vSym)
		case "SVar":
			asign := NewAsign()
			asign.AddSym(sym)

			t, isEqual, err := p.match(fxlex.Assignation)
			if err != nil {
				return err
			} else if !isEqual {
				p.errorf("%s:%d: syntax error: bad statement",
					p.l.GetFilename(), p.l.GetLineNumber())
				err = p.l.SkipUntil(fxlex.Semicolon)
				if err != nil {
					return err
				}
			} else {
				p.pushTrace(fmt.Sprintf("%s", t))
				p.popTrace()
			}

			expr, err := p.Expr(defRbp - 1)
			if err != nil {
				return err
			}
			asign.AddValue(expr)

			t, isSemicolon, err := p.match(fxlex.Semicolon)
			if err != nil {
				return err
			} else if !isSemicolon {
				p.errorf("%s:%d: syntax error: bad statement",
					p.l.GetFilename(), p.l.GetLineNumber())
				err = p.l.SkipUntilAndLex(fxlex.Semicolon)
				return err
			} else {
				p.pushTrace(fmt.Sprintf("%s", t))
				p.popTrace()
			}

			stm.AddAsign(asign)
		default:
			p.errorf("%s:%d: syntax error: symbol %s not expected",
				p.l.GetFilename(), p.l.GetLineNumber(), tokID.GetLexeme())
		}
	case fxlex.TokKey:
		t, err = p.l.Lex()

		switch t.GetLexeme() {
		case "iter":
			p.pushTrace(fmt.Sprintf("Key %s", t))
			p.popTrace()

			iter := NewIter()
			if err := p.Iter(iter); err != nil {
				return err
			}

			stm.AddIter(iter)
		case "if":
			p.pushTrace(fmt.Sprintf("Key %s", t))
			p.popTrace()

			nodeIf := NewNodeIf()
			if err := p.NodeIf(nodeIf); err != nil {
				return err
			}

			stm.AddNodeIf(nodeIf)
		default:
			p.errorf("%s:%d: syntax error: keyword unexpected",
				p.l.GetFilename(), p.l.GetLineNumber())
			err = p.l.SkipUntil(fxlex.TokLPar, fxlex.TokRPar, fxlex.TokComma)
			if err != nil {
				return err
			}
		}
	case fxlex.TokLCurl:
		t, _ = p.l.Lex()

		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()

		inner_body := NewBody()
		if err := p.Body(inner_body); err != nil {
			return err
		}

		t, isRCurl, err := p.match(fxlex.TokRCurl)
		if err != nil {
			return err
		} else if !isRCurl {
			p.errorf("%s:%d: syntax error: bad statement",
				p.l.GetFilename(), p.l.GetLineNumber())
			err = p.l.SkipUntilAndLex(fxlex.TokRCurl)
			return err
		} else {
			p.pushTrace(fmt.Sprintf("%s", t))
			p.popTrace()
		}

		stm.AddBody(inner_body)
	default:
		return err
	}

	body.AddStm(stm)

	return p.Body(body)
}

// <CALL> ::= ')' ';' |
//            <ARGS_LIST> ')' ';'
func (p *Parser) Call(call *Call) error {
	p.pushTrace("Call")
	defer p.popTrace()

	t, isRPar, err := p.match(fxlex.TokRPar)
	if err != nil {
		return err
	} else if !isRPar {
		err := p.ArgsList(call)
		if err != nil {
			return err
		}

		t, isRPar, err := p.match(fxlex.TokRPar)
		if err != nil {
			return err
		} else if !isRPar {
			p.errorf("%s:%d: syntax error: bad statement",
				p.l.GetFilename(), p.l.GetLineNumber())
			err = p.l.SkipUntil(fxlex.Semicolon)
			if err != nil {
				return err
			}
		} else {
			p.pushTrace(fmt.Sprintf("%s", t))
			p.popTrace()
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	t, isSemicolon, err := p.match(fxlex.Semicolon)
	if err != nil {
		return err
	} else if !isSemicolon {
		p.errorf("%s:%d: syntax error: bad statement",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntilAndLex(fxlex.Semicolon)
		return err
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	return err
}

// <ARGS_LIST> ::= <EXPR> <ARGS>
func (p *Parser) ArgsList(call *Call) error {
	p.pushTrace("ArgsList")
	defer p.popTrace()

	arg, err := p.Expr(defRbp - 1)
	if err != nil {
		return err
	}
	call.AddArg(arg)

	return p.Args(call)
}

// <ARGS> ::= ',' <EXPR> <ARGS>
//            <Empty>
func (p *Parser) Args(call *Call) error {
	p.pushTrace("Args")
	defer p.popTrace()

	t, isComma, err := p.match(fxlex.TokComma)
	if err != nil || !isComma {
		return err
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	arg, err := p.Expr(defRbp - 1)
	if err != nil {
		return err
	}
	call.AddArg(arg)

	return p.Args(call)
}

// <ITER> ::= '(' id ':=' <EXPR> ',' <EXPR> ',' <EXPR> ')' '{' <BODY> '}'
func (p *Parser) Iter(iter *Iter) error {
	p.pushTrace("Iter")
	defer p.popTrace()

	p.stkEnv.PushEnv()
	defer p.stkEnv.PopEnv()

	t, isLPar, err := p.match(fxlex.TokLPar)
	if err != nil {
		return err
	} else if !isLPar {
		p.errorf("%s:%d: syntax error: iter (bad statement)",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokLCurl, fxlex.TokRCurl, fxlex.TokRPar, fxlex.TokComma)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	t, isID, err := p.match(fxlex.TokID)
	if err != nil {
		return err
	} else if !isID {
		p.errorf("%s:%d: syntax error: iter (bad statement)",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokLCurl, fxlex.TokRCurl, fxlex.TokRPar, fxlex.TokComma)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("ID %s", t))
		p.popTrace()
	}

	varControl, err := p.stkEnv.NewSym(t.GetLexeme(), fxsym.SVar)
	if err != nil {
		p.errorf("%s:%d: syntax error: %s (%s)",
			p.l.GetFilename(), p.l.GetLineNumber(), err, t.GetLexeme())
	} else {
		varControl.AddTokKind(t.GetTokType())
		varControl.AddPlace(p.l.GetFilename(), p.l.GetLineNumber())
		varControl.SetType(Types[TInt].id)
	}

	iter.AddVarControl(varControl)

	t, isDeclaration, err := p.match(fxlex.Declaration)
	if err != nil {
		return err
	} else if !isDeclaration {
		p.errorf("%s:%d: syntax error: iter (bad statement)",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokLCurl, fxlex.TokRCurl, fxlex.TokRPar, fxlex.TokComma)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	e, err := p.Expr(defRbp - 1)
	if err != nil {
		return err
	}

	iter.AddStart(e)

	t, isComma, err := p.match(fxlex.TokComma)
	if err != nil {
		return err
	} else if !isComma {
		p.errorf("%s:%d: syntax error: iter (bad statement)",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokLCurl, fxlex.TokRCurl, fxlex.TokRPar, fxlex.TokComma)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	e, err = p.Expr(defRbp - 1)
	if err != nil {
		return err
	}

	iter.AddEnd(e)

	t, isComma, err = p.match(fxlex.TokComma)
	if err != nil {
		return err
	} else if !isComma {
		p.errorf("%s:%d: syntax error: iter (bad statement)",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokLCurl, fxlex.TokRCurl, fxlex.TokRPar)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	e, err = p.Expr(defRbp - 1)
	if err != nil {
		return err
	}

	iter.AddStep(e)

	t, isRPar, err := p.match(fxlex.TokRPar)
	if err != nil {
		return err
	} else if !isRPar {
		p.errorf("%s:%d: syntax error: iter (bad statement)",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokLCurl, fxlex.TokRCurl)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	t, isLCurl, err := p.match(fxlex.TokLCurl)
	if err != nil {
		return err
	} else if !isLCurl {
		p.errorf("%s:%d: syntax error: iter (bad statement)",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokRCurl)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	if err = p.Body(iter.body); err != nil {
		return err
	}

	t, isRCurl, err := p.match(fxlex.TokRCurl)
	if err != nil {
		return err
	} else if !isRCurl {
		p.errorf("%s:%d: syntax error: iter (bad statement)",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntilAndLex(fxlex.TokRCurl)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	return err
}

// <IF> ::= '(' <EXPR> ')' '{' <BODY> '}' <ELSE>
func (p *Parser) NodeIf(nodeIf *NodeIf) error {
	p.pushTrace("If")
	defer p.popTrace()

	t, isLPar, err := p.match(fxlex.TokLPar)
	if err != nil {
		return err
	} else if !isLPar {
		p.errorf("%s:%d: syntax error: if (bad statement)",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokLCurl, fxlex.TokRCurl, fxlex.TokRPar)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	e, err := p.Expr(defRbp - 1)
	if err != nil {
		return err
	}

	nodeIf.AddCond(e)

	t, isRPar, err := p.match(fxlex.TokRPar)
	if err != nil {
		return err
	} else if !isRPar {
		p.errorf("%s:%d: syntax error: if (bad statement)",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokLCurl, fxlex.TokRCurl)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	t, isLCurl, err := p.match(fxlex.TokLCurl)
	if err != nil {
		return err
	} else if !isLCurl {
		p.errorf("%s:%d: syntax error: if (bad statement)",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokRCurl)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	if err = p.Body(nodeIf.body); err != nil {
		return err
	}

	t, isRCurl, err := p.match(fxlex.TokRCurl)
	if err != nil {
		return err
	} else if !isRCurl {
		p.errorf("%s:%d: syntax error: if (bad statement)",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntilAndLex(fxlex.TokRCurl)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	return p.Else(nodeIf)
}

// <ELSE> ::= 'else' '{' <BODY> '}' |
//            <Empty>
func (p *Parser) Else(nodeIf *NodeIf) error {
	p.pushTrace("Else")
	defer p.popTrace()

	tokElse, err := p.l.Peek()
	if err != nil || tokElse.GetLexeme() != "else" {
		return err
	}

	p.l.Lex()

	t, isLCurl, err := p.match(fxlex.TokLCurl)
	if err != nil {
		return err
	} else if !isLCurl {
		p.errorf("%s:%d: syntax error: if (bad statement)",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokRCurl)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	nodeIf.bodyElse = NewBody()
	if err = p.Body(nodeIf.bodyElse); err != nil {
		return err
	}

	t, isRCurl, err := p.match(fxlex.TokRCurl)
	if err != nil {
		return err
	} else if !isRCurl {
		p.errorf("%s:%d: syntax error: if (bad statement)",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntilAndLex(fxlex.TokRCurl)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	return err
}
