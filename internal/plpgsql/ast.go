package plpgsql

import pg_query "github.com/gosuda/pg18_query_go/internal/pgproto"

type Expression struct {
	Query     string
	ParseMode int
	SQL       *pg_query.ParseResult
}

type DatumKind uint8

const (
	Scalar DatumKind = iota
	Row
	Record
	RecordField
	Promise
)

type Field struct {
	Name  string
	Varno int
}
type Datum struct {
	Kind          DatumKind
	Number        int
	Name          string
	Line          int
	Type          string
	Constant      bool
	NotNull       bool
	Default       *Expression
	CursorQuery   *Expression
	CursorArgRow  int
	CursorOptions int
	Fields        []Field
	Parent        int
}

type Function struct {
	NewVarno int
	OldVarno int
	Datums   []*Datum
	Action   *Block
}

type Parameter struct {
	Name, Type string
	Output     bool
	Input      bool
}
type Config struct {
	Name       string
	Parameters []Parameter
	ReturnType string
	ReturnsSet bool
	Procedure  bool
	Inline     bool
}

// SQLParser validates embedded SQL and retains the native SQL tree in the AST.
type SQLParser func(string, int) (*pg_query.ParseResult, error)

type Statement interface{ statement() }
type Base struct{ Line int }

func (Base) statement() {}

type Block struct {
	Base
	Label      string
	Body       []Statement
	Exceptions []Exception
}
type Exception struct {
	Conditions []string
	Action     []Statement
}
type Assign struct {
	Base
	Varno int
	Expr  *Expression
}
type If struct {
	Base
	Cond   *Expression
	Then   []Statement
	Elsifs []Elsif
	Else   []Statement
}
type Elsif struct {
	Line int
	Cond *Expression
	Body []Statement
}
type Case struct {
	Base
	Test     *Expression
	Varno    int
	Whens    []When
	HaveElse bool
	Else     []Statement
}
type When struct {
	Line int
	Expr *Expression
	Body []Statement
}
type Loop struct {
	Base
	Kind                                      string
	Label                                     string
	Var                                       *Datum
	Varno                                     int
	Curvar                                    int
	Slice                                     int
	Reverse                                   bool
	Cond, Lower, Upper, Step, Query, Argquery *Expression
	Params                                    []*Expression
	Body                                      []Statement
}
type Exit struct {
	Base
	IsExit bool
	Label  string
	Cond   *Expression
}
type Return struct {
	Base
	Kind                  string
	Expr, Query, Dynquery *Expression
	Params                []*Expression
}
type Raise struct {
	Base
	Level     int
	Condition string
	Message   *string
	Params    []*Expression
	Options   []RaiseOption
}
type RaiseOption struct {
	Kind int
	Expr *Expression
}
type Assert struct {
	Base
	Cond, Message *Expression
}
type Execute struct {
	Base
	Dynamic      bool
	Query        *Expression
	Into, Strict bool
	Target       *Datum
	Params       []*Expression
}
type Diagnostic struct {
	Kind   string
	Target int
}
type GetDiagnostics struct {
	Base
	Stacked bool
	Items   []Diagnostic
}
type Open struct {
	Base
	Curvar, Options      int
	Args, Query, Dynamic *Expression
	Params               []*Expression
}
type Fetch struct {
	Base
	Target            *Datum
	Curvar, Direction int
	Count             int64
	Expr              *Expression
	Move, Multiple    bool
}
type Close struct {
	Base
	Curvar int
}
type Perform struct {
	Base
	Expr *Expression
}
type Call struct {
	Base
	Expr   *Expression
	IsCall bool
	Target *Datum
}
type Transaction struct {
	Base
	Rollback, Chain bool
}
