
package gotypes

type Ityp int

type Ptyp *string

type PDtyp *Ptyp

type PMtyp *map[string]Ityp

type Mtyp map[string]int

type Sltyp []float32

type Artyp [20]float64

type Sttyp struct {
	Anon1
	ki.Node
	FieldA int
	FieldB Sltyp
	FieldC, FieldD string
}

type Sl2typ []Ityp

type Iftyp interface {
	ki.Ki
	LocalIface
	MethA(Ityp a, b) Ptyp
	MethB(Ityp a, b) Ptyp
}

type Dtyp Sttyp

type DPtyp ki.Ki

type Futyp func(ab, bb string, Sttyp, int, ki.Ki) PMtyp

type Fustyp func()

type Furvtyp func() (Ityp, int, string)

type Funrtyp func() (err error)

var Ivar Ityp

var Bvar int

var Svar Sttyp

var Svr2, Svr3 Sttyp

var (
	Mvar map[string]float32
	Avar []ki.Ki
)

func (st *Sttyp) Method1(para int, meter string) {
	st.FieldA = para
	st.FieldC = meter
	
	nwvar := st.FieldA
	var nmvar Mtyp
}

func (st *Sttyp) Method2(para int, meter string) (bool, int) {
	st.FieldA = para
	st.FieldC = meter
	return false, -1
}

func (st *Sttyp) MethodNP() {
	st.FieldA = 22
	st.FieldC = "be"
}


