package fxparse

import (
	"errors"
	"fmt"
	"fxlex"
	"os"
	"strings"
)

var precTab = map[rune]int{
	')':          1,
	'|':          10,
	'&':          10,
	'^':          10,
	'!':          20,
	'<':          30,
	'>':          30,
	fxlex.TokGTE: 30,
	fxlex.TokLTE: 30,
	'+':          40,
	'-':          40,
	'*':          50,
	'/':          50,
	'%':          50,
	fxlex.TokPow: 60,
	'(':          70,
}

var leftTab = map[rune]bool{
	fxlex.TokPow: true,
}
var unaryTab = map[rune]bool{
	'+': true,
	'-': true,
	'(': true,
	'!': true,
	'^': true,
}

//no left context, null-denotation: nud
func (p *Parser) Nud(tok fxlex.Token) (expr *Expr, err error) {
	var rExpr *Expr
	var rbp int
	p.dPrintf("Nud:  %d, %s \n", rbp, tok)
	if tok.GetTokType() == fxlex.TokLPar { //special unary, parenthesis
		expr, err = p.Expr(rbp)
		if err != nil {
			return nil, err
		}
		if _, isClosed, err := p.match(fxlex.TokRPar); err != nil {
			return nil, err
		} else if !isClosed {
			return nil, errors.New("unmatched parenthesis")
		}
		return expr, nil
	}
	expr = NewExpr(tok)
	rbp = bindPow(tok)
	rTok := rune(tok.GetTokType())
	if rbp != defRbp { //regular unary operators
		if !unaryTab[rTok] {
			errs := fmt.Sprintf("%s  is not unary", tok.GetType())
			return nil, errors.New(errs)
		}
		rExpr, err = p.Expr(rbp)
		if rExpr == nil {
			return nil, errors.New("unary operator without operand")
		}
		expr.ERight = rExpr
	}
	return expr, nil
}

//left context, left-denotation: led
func (p *Parser) Led(left *Expr, tok fxlex.Token) (expr *Expr, err error) {
	var rbp int
	expr = NewExpr(tok)
	expr.ELeft = left
	rbp = bindPow(tok)
	if isleft := leftTab[rune(tok.GetTokType())]; isleft {
		rbp -= 1
	}
	p.dPrintf("Led: %d, {{%s}} %s \n", rbp, left, tok)
	rExpr, err := p.Expr(rbp)
	if err != nil {
		return nil, err
	}
	if rExpr == nil {
		errs := fmt.Sprintf("missing operand for %s\n", tok.GetType())
		return nil, errors.New(errs)
	}
	expr.ERight = rExpr
	return expr, nil
}

const defRbp = 0

func bindPow(tok fxlex.Token) int {
	if rbp, ok := precTab[rune(tok.GetTokType())]; ok {
		return rbp
	}
	return defRbp
}

func (p *Parser) Expr(rbp int) (expr *Expr, err error) {
	var left *Expr

	s := fmt.Sprintf("Expr: %d", rbp)
	p.pushTrace(s)
	defer p.popTrace()

	tok, err := p.l.Peek()
	if err != nil {
		return expr, err
	}
	p.dPrintf("expr: Nud Lex: %s", tok)

	if tok.GetTokType() == fxlex.RuneEOF {
		return expr, nil
	}
	p.l.Lex() //already peeked
	if left, err = p.Nud(tok); err != nil {
		return nil, err
	}
	expr = left
	for {
		tok, err := p.l.Peek()
		if err != nil {
			return expr, err
		}
		if tok.GetTokType() == fxlex.RuneEOF || tok.GetTokType() == fxlex.TokRPar || tok.GetTokType() == fxlex.TokComma || tok.GetTokType() == fxlex.Semicolon {
			return expr, nil
		}
		if bindPow(tok) <= rbp {
			p.dPrintf("Not enough binding: %d <= %d, %s\n", bindPow(tok), rbp, tok)
			return left, nil
		}
		p.l.Lex() //already peeked
		p.dPrintf("expr: led Lex: %s", tok)
		if left, err = p.Led(left, tok); err != nil {
			return expr, err
		}
		expr = left
	}
}

func (p *Parser) dPrintf(format string, a ...interface{}) {
	if DebugParser {
		tabs := strings.Repeat("\t", p.depth)
		format = fmt.Sprintf("%s%s", tabs, format)
		fmt.Fprintf(os.Stderr, format, a...)
	}
}
