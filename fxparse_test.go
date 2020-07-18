package fxparse_test

import (
	"bufio"
	"fxlex"
	. "fxparse"
	"strings"
	"testing"
)

var exampleFile = `//basic types bool, int (64 bits), Coord(int x, int y)
//literals of type int are 2, 3, or 0x2dfadfd
//literals of Coord are [3,4] [0x46,4]
//literals of bool are True, False
//operators of int are + - * / ** > >= < <=
//operators of int are %
//operators of bool are | & ! ^
//precedence is like in C, with ** having the
//same precedence as sizeof (not present in fx)

//builtins
//circle(p, 2, 0x1100001f);
//	at point p, int radius r, color: transparency and rgb
//rect(p, α, col);
//	at point p, int angle (degrees),
//	color: transparency (0-100) and rgb

//macro definition
func line(int x, int y){
	iter (i := 0, x, 1){	//declares the variable only in the loop
		circle(2, 3, y, 5);
	}
}

//macro entry
func main(){
  iter (i := 0, 3, 1){
    rect(i, i, 3, 0xff);
  }
  iter (j := 0, 8, 2){ // loops 0 2 4 6 8
    rect(j, j, 8, 0xff);
  }
  circle(4, 5, 2, 0x11000011);
}
`

func newTestParser(t *testing.T, text string) (p *Parser) {
	reader := bufio.NewReader(strings.NewReader(text))
	l, err := fxlex.NewLexer(reader, "test")
	if err != nil {
		t.Fatalf("lexer instantiation failed")
	}
	p, _ = NewParser(l)

	return p
}

func TestParse(t *testing.T) {
	p := newTestParser(t, exampleFile)
	DebugParser = false
	fxlex.DebugLexer = false

	if err := p.Parse(); err != nil {
		t.Errorf("TestParse failed")
	}
}
