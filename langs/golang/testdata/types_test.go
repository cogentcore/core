
package gotypes

type Ityp int

type Ptyp *string

type PDtyp *Ptyp

type PMtyp *map[string]Ityp

type Mtyp map[string]int

type Sltyp []float32

type Artyp [20]float64

type Sttyp struct {
	A int
	B Sltyp
}

type Sl2typ []Ityp

type Ityp interface {
	MethA(Ityp a, b) Ptyp
	MethB(Ityp a, b) Ptyp
}

type Dtyp Sttyp

type DPtyp ki.Ki

