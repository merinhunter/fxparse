package fxparse

import (
	"fmt"
	"fxlex"
	"os"
	"strings"
)

const maxErrors = 5

var DebugParser bool = false

type Parser struct {
	l     *fxlex.Lexer
	nErr  int
	depth int
	//stkEnv fxsym.StkEnv
}

func NewParser(l *fxlex.Lexer) (p *Parser, err error) {
	//p = &Parser{l, 0, 0, nil}
	p = &Parser{l, 0, 0}

	// p.stkEnv.PushEnv()
	// p.initSyms()
	// p.stkEnv.PushEnv()

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

		prog.AddFunc(f)

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

	f = NewFunc()

	if err := p.Head(f.Head()); err != nil {
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

	if err := p.Body(f.Body()); err != nil {
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

		head.AddID(t.GetLexeme())
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

	param := NewVar()

	t, isTypeID, err := p.match(fxlex.TokID)
	if err != nil || !isTypeID {
		return err
	}

	p.pushTrace(fmt.Sprintf("TypeID %s", t))
	p.popTrace()

	param.AddVarType(t.GetTokType())

	t, isID, err := p.match(fxlex.TokID)
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
		p.pushTrace(fmt.Sprintf("ID %s", t))
		p.popTrace()

		param.AddID(t.GetLexeme())
	}

	head.AddParam(param)

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
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	param := NewVar()

	t, isTypeID, err := p.match(fxlex.TokID)
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
		p.pushTrace(fmt.Sprintf("TypeID %s", t))
		p.popTrace()

		param.AddVarType(t.GetTokType())
	}

	t, isID, err := p.match(fxlex.TokID)
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
		p.pushTrace(fmt.Sprintf("ID %s", t))
		p.popTrace()

		param.AddID(t.GetLexeme())
	}

	head.AddParam(param)

	return p.Prms(head)
}

// <BODY> ::= id '(' <CALL> <BODY> |
//            'iter' <ITER> <BODY> |
//            '{' <BODY> '}' |
//            <Empty>
func (p *Parser) Body(body *Body) error {
	// maybe this function should be divided in 2, one for the statements and the other for the body
	p.pushTrace("Body")
	defer p.popTrace()

	t, err := p.l.Peek()
	if err != nil {
		return err
	}

	//p.stkEnv.PushEnv()
	//defer p.stkEnv.PopEnv()

	//b = &fxsym.Body{}

	stm := NewStatement()

	switch t.GetTokType() {
	case fxlex.TokID:
		tokID, err := p.l.Lex()
		p.pushTrace(fmt.Sprintf("ID %s", tokID))
		p.popTrace()

		call := NewCall(tokID.GetLexeme())

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

	//call = &fxsym.Call{ID: id}

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

	arg, err := p.Expr()
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

	arg, err := p.Expr()
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

	//p.stkEnv.PushEnv()
	//defer p.stkEnv.PopEnv()

	//iter = &fxsym.Iter{}

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

		varControl := NewVar()
		varControl.AddID(t.GetLexeme())
		varControl.AddVarType(fxlex.TokID)

		iter.AddVarControl(varControl)
	}

	/*start, err := p.stkEnv.NewSym(t.GetLexeme(), fxsym.SVar)
	if err != nil {
		return nil, err
	}
	start.Decl = &fxsym.Decl{ID: t.GetLexeme()}*/

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

	e, err := p.Expr()
	if err != nil {
		return err
	}

	iter.AddStart(e)

	//start.Decl.Val = e
	//iter.Start = start

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

	e, err = p.Expr()
	if err != nil {
		return err
	}

	iter.AddEnd(e)
	//iter.End = e

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

	e, err = p.Expr()
	if err != nil {
		return err
	}

	iter.AddStep(e)
	//iter.Step = e

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

	if err = p.Body(iter.Body()); err != nil {
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

// <EXPR> ::= <ATOM>
func (p *Parser) Expr() (e *Expr, err error) {
	p.pushTrace("Expr")
	defer p.popTrace()

	tok, err := p.Atom()
	if err != nil {
		return nil, err
	}

	return NewExpr(tok), nil
}

// <ATOM> ::= id |
//						num |
//						bool
//func (p *Parser) Atom() (s *fxsym.Sym, err error) {
func (p *Parser) Atom() (t fxlex.Token, err error) {
	p.pushTrace("Atom")
	defer p.popTrace()

	t, err = p.l.Peek()
	if err != nil {
		return t, err
	}

	switch t.GetTokType() {
	case fxlex.TokID:
		t, err = p.l.Lex()
		if err != nil {
			return t, err
		}

		p.pushTrace(fmt.Sprintf("ID %s", t))
		defer p.popTrace()

		/*s = p.stkEnv.GetSym(t.GetLexeme())
		if s == nil {
			return nil, errors.New("ID not found: " + t.GetLexeme())
		}*/
	case fxlex.TokIntLit:
		t, err = p.l.Lex()
		if err != nil {
			return t, err
		}

		p.pushTrace(fmt.Sprintf("Num %s", t))
		defer p.popTrace()

		//s = &fxsym.Sym{IntVal: t.GetValue()}
	case fxlex.TokBoolLit:
		t, err = p.l.Lex()
		if err != nil {
			return t, err
		}

		p.pushTrace(fmt.Sprintf("Bool %s", t))
		defer p.popTrace()

		//s = &fxsym.Sym{BoolVal: t.GetValue() != 0}
	default:
		p.errorf("%s:%d: syntax error: expected id, number or bool, found %s",
			p.l.GetFilename(), p.l.GetLineNumber(), t.GetTokType())
	}

	return t, err
}
