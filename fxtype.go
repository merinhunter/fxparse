package fxparse

const (
	TUndef = iota
	TInt
	TBool
	TCoord
	NTypes
)

var typeNames = []string{
	TUndef: "undef",
	TInt:   "int",
	TBool:  "bool",
	TCoord: "Coord",
}

type Type struct {
	id int
}

var Types = []*Type{
	TUndef: &Type{TUndef},
	TInt:   &Type{TInt},
	TBool:  &Type{TBool},
	TCoord: &Type{TCoord},
}

func (tp *Type) String() string {
	if tp == nil || tp.id < TUndef || tp.id >= NTypes {
		return "unktype"
	}
	return typeNames[tp.id]
}
