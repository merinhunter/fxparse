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
}

func NewParser(l *fxlex.Lexer) *Parser {
	return &Parser{l, 0, 0}
}

func (p *Parser) Parse() error {
	p.pushTrace("Parse")
	defer p.popTrace()

	if err := p.Prog(); err != nil {
		return err
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
func (p *Parser) Prog() error {
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

		if err := p.Func(); err != nil {
			return err
		}

		return p.Prog()
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
func (p *Parser) Func() error {
	p.pushTrace("Func")
	defer p.popTrace()

	if err := p.Head(); err != nil {
		return err
	}

	t, isLCurl, err := p.match(fxlex.TokLCurl)
	if err != nil {
		return err
	} else if !isLCurl {
		p.errorf("%s:%d: syntax error: macro bad definition",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntil(fxlex.TokLCurl, fxlex.TokRCurl)
		if err != nil {
			return err
		}
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	if err := p.Body(); err != nil {
		return err
	}

	t, isRCurl, err := p.match(fxlex.TokRCurl)
	if err != nil {
		return err
	} else if !isRCurl {
		p.errorf("%s:%d: syntax error: macro bad definition",
			p.l.GetFilename(), p.l.GetLineNumber())
		err = p.l.SkipUntilAndLex(fxlex.TokRCurl)
		return err
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	return err
}

// <HEAD> ::= id '(' <FORMAL_PRMS> ')'
func (p *Parser) Head() error {
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

	if err := p.FormalPrms(); err != nil {
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
func (p *Parser) FormalPrms() error {
	p.pushTrace("FormalPrms")
	defer p.popTrace()

	t, isTypeID, err := p.match(fxlex.TokID)
	if err != nil || !isTypeID {
		return err
	} else {
		p.pushTrace(fmt.Sprintf("TypeID %s", t))
		p.popTrace()
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
	}

	return p.Prms()
}

// <PRMS> ::= ',' type_id id <PRMS>
//            <Empty>
func (p *Parser) Prms() error {
	p.pushTrace("Prms")
	defer p.popTrace()

	t, isComma, err := p.match(fxlex.TokComma)
	if err != nil || !isComma {
		return err
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

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
	}

	return p.Prms()
}

// <BODY> ::= id '(' <CALL> <BODY> |
//            'iter' <ITER> <BODY> |
//            '{' <BODY> '}' |
//            <Empty>
func (p *Parser) Body() error {
	p.pushTrace("Body")
	defer p.popTrace()

	t, err := p.l.Peek()
	if err != nil {
		return err
	}

	switch t.GetTokType() {
	case fxlex.TokID:
		t, err = p.l.Lex()
		p.pushTrace(fmt.Sprintf("ID %s", t))
		p.popTrace()

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

		if err := p.Call(); err != nil {
			return err
		}

		return p.Body()
	case fxlex.TokKey:
		t, err = p.l.Lex()

		switch t.GetLexeme() {
		case "iter":
			p.pushTrace(fmt.Sprintf("Key %s", t))
			p.popTrace()

			if err := p.Iter(); err != nil {
				return err
			}

			return p.Body()
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

		if err := p.Body(); err != nil {
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
	default:
		return err
	}

	return err
}

// <CALL> ::= ')' ';' |
//            <ARGS_LIST> ')' ';'
func (p *Parser) Call() error {
	p.pushTrace("Call")
	defer p.popTrace()

	t, isRPar, err := p.match(fxlex.TokRPar)
	if err != nil {
		return err
	} else if !isRPar {
		if err := p.ArgsList(); err != nil {
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
func (p *Parser) ArgsList() error {
	p.pushTrace("ArgsList")
	defer p.popTrace()

	if err := p.Expr(); err != nil {
		return err
	}

	return p.Args()
}

// <ARGS> ::= ',' <EXPR> <ARGS>
//            <Empty>
func (p *Parser) Args() error {
	p.pushTrace("Args")
	defer p.popTrace()

	t, isComma, err := p.match(fxlex.TokComma)
	if err != nil || !isComma {
		return err
	} else {
		p.pushTrace(fmt.Sprintf("%s", t))
		p.popTrace()
	}

	if err := p.Expr(); err != nil {
		return err
	}

	return p.Args()
}

// <ITER> ::= '(' id ':=' <EXPR> ',' <EXPR> ',' <EXPR> ')' '{' <BODY> '}'
func (p *Parser) Iter() error {
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

	if err := p.Expr(); err != nil {
		return err
	}

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

	if err := p.Expr(); err != nil {
		return err
	}

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

	if err := p.Expr(); err != nil {
		return err
	}

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

	if err := p.Body(); err != nil {
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
func (p *Parser) Expr() error {
	p.pushTrace("Expr")
	defer p.popTrace()

	return p.Atom()
}

// <ATOM> ::= id |
//						num |
//						bool
func (p *Parser) Atom() error {
	p.pushTrace("Atom")
	defer p.popTrace()

	t, err := p.l.Peek()
	if err != nil {
		return err
	}

	switch t.GetTokType() {
	case fxlex.TokID:
		t, err = p.l.Lex()
		p.pushTrace(fmt.Sprintf("ID %s", t))
		defer p.popTrace()
	case fxlex.TokIntLit:
		t, err = p.l.Lex()
		p.pushTrace(fmt.Sprintf("Num %s", t))
		defer p.popTrace()
	case fxlex.TokBoolLit:
		t, err = p.l.Lex()
		p.pushTrace(fmt.Sprintf("Bool %s", t))
		defer p.popTrace()
	default:
		p.errorf("%s:%d: syntax error: expected id, number or bool, found %s",
			p.l.GetFilename(), p.l.GetLineNumber(), t.GetTokType())
	}

	return err
}
