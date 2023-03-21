package parser

type Token int

const (
	ILLEGAL Token = iota
	EOF

	// Keywords
	LET
	IN

	// Punctuation
	EQUALS
	R_PAREN
	L_PAREN
	R_BRACKET
	L_BRACKET
	R_CURL
	L_CURL
	STRING
	COMMENT
	COLON
	COMMA

	// Functions
	F_SOLVE
	F_SOLVELEGACY

	IDENTIFIER
)

var keywordTokens = map[string]Token{
	"let":          LET,
	"in":           IN,
	"solve":        F_SOLVE,
	"solve_legacy": F_SOLVELEGACY,
}

var tokenNames = map[Token]string{
	ILLEGAL:       "ILLEGAL",
	EOF:           "EOF",
	LET:           "LET",
	IN:            "IN",
	EQUALS:        "EQUALS",
	R_PAREN:       "R_PAREN",
	L_PAREN:       "L_PAREN",
	R_BRACKET:     "R_BRACKET",
	L_BRACKET:     "L_BRACKET",
	STRING:        "STRING",
	COMMENT:       "COMMENT",
	COLON:         "COLON",
	COMMA:         "COMMA",
	F_SOLVE:       "F_SOLVE",
	F_SOLVELEGACY: "F_SOLVELEGACY",
	IDENTIFIER:    "IDENTIFIER",
}

func (t Token) String() string {
	return tokenNames[t]
}
