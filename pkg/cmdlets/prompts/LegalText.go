package prompts

type LegalText interface {
	GetLegalText() (string, error)
}

