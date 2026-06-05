package ssair

type Program struct {
	Funcs []Function `json:"funcs"`
}

type Function struct {
	Name       string  `json:"name"`
	ReturnType Type    `json:"return_type"`
	Values     []Value `json:"values,omitempty"`
	Blocks     []Block `json:"blocks,omitempty"`
}

type Type string

const (
	TypeVoid   Type = "void"
	TypeI32    Type = "i32"
	TypeBool   Type = "bool"
	TypePtr    Type = "ptr"
	TypeEffect Type = "effect"
	TypeString Type = "string"
)

type ValueID string

type Value struct {
	ID     ValueID `json:"id"`
	Type   Type    `json:"type"`
	Origin string  `json:"origin,omitempty"`
}

type Block struct {
	ID     string     `json:"id"`
	Entry  bool       `json:"entry,omitempty"`
	Params []ValueID  `json:"params,omitempty"`
	Instrs []Instr    `json:"instrs,omitempty"`
	Term   Terminator `json:"term"`
}

type OpKind string

const (
	OpConstI32     OpKind = "const_i32"
	OpAddI32       OpKind = "add_i32"
	OpSubI32       OpKind = "sub_i32"
	OpMulI32       OpKind = "mul_i32"
	OpDivI32       OpKind = "div_i32"
	OpModI32       OpKind = "mod_i32"
	OpCmpEqI32     OpKind = "cmp_eq_i32"
	OpCmpLtI32     OpKind = "cmp_lt_i32"
	OpCmpGtI32     OpKind = "cmp_gt_i32"
	OpCmpGeI32     OpKind = "cmp_ge_i32"
	OpCmpLeI32     OpKind = "cmp_le_i32"
	OpCmpNeI32     OpKind = "cmp_ne_i32"
	OpNegI32       OpKind = "neg_i32"
	OpCall         OpKind = "call"
	OpIndexLoadI32 OpKind = "index_load_i32"
	OpOpaque       OpKind = "opaque"
)

type Instr struct {
	ID        string    `json:"id"`
	Kind      OpKind    `json:"kind"`
	Result    ValueID   `json:"result,omitempty"`
	Type      Type      `json:"type,omitempty"`
	Args      []ValueID `json:"args,omitempty"`
	Imm       int32     `json:"imm,omitempty"`
	Call      string    `json:"call,omitempty"`
	EffectIn  ValueID   `json:"effect_in,omitempty"`
	EffectOut ValueID   `json:"effect_out,omitempty"`
	ProofID   string    `json:"proof_id,omitempty"`
	Note      string    `json:"note,omitempty"`
}

type TermKind string

const (
	TermInvalid TermKind = ""
	TermReturn  TermKind = "return"
	TermBranch  TermKind = "branch"
	TermCondBr  TermKind = "cond_br"
)

type Terminator struct {
	Kind        TermKind  `json:"kind"`
	Value       ValueID   `json:"value,omitempty"`
	Target      string    `json:"target,omitempty"`
	Args        []ValueID `json:"args,omitempty"`
	Cond        ValueID   `json:"cond,omitempty"`
	IfTrue      string    `json:"if_true,omitempty"`
	IfTrueArgs  []ValueID `json:"if_true_args,omitempty"`
	IfFalse     string    `json:"if_false,omitempty"`
	IfFalseArgs []ValueID `json:"if_false_args,omitempty"`
}
