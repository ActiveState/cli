package parser

type Token int

const (
	ILLEGAL Token = iota
	EOF

	// Keywords
	LET
	IN

	// Punctuation
	COLON
	R_BRACKET
	L_BRACKET
	R_CURL
	L_CURL
	QUOTATION
	COMMA

	// Functions
	F_SOLVE
	F_SOLVELEGACY
	F_MERGE

	STRING

	IDENTIFIER

	NULL
)

var keywordTokens = map[string]Token{
	"let":          LET,
	"in":           IN,
	"solve":        F_SOLVE,
	"solve_legacy": F_SOLVELEGACY,
	"merge":        F_MERGE,
}

var tokenNames = map[Token]string{
	ILLEGAL:       "ILLEGAL",
	EOF:           "EOF",
	LET:           "LET",
	IN:            "IN",
	R_BRACKET:     "R_BRACKET",
	L_BRACKET:     "L_BRACKET",
	R_CURL:        "R_CURL",
	L_CURL:        "L_CURL",
	QUOTATION:     "STRING",
	COLON:         "COLON",
	COMMA:         "COMMA",
	F_SOLVE:       "F_SOLVE",
	F_SOLVELEGACY: "F_SOLVELEGACY",
	F_MERGE:       "F_MERGE",
	STRING:        "STRING",
	IDENTIFIER:    "IDENTIFIER",
	NULL:          "NULL",
}

func (t Token) String() string {
	return tokenNames[t]
}
