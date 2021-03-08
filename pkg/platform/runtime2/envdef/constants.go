package envdef

// Constants is a map of constants that are being expanded in environment variables and file transformations to their installation-specific values
type Constants map[string]string

// NewConstants initializes a new map of constants that will need to be set to installation-specific values
// Currently it only has one field `INSTALLDIR`
func NewConstants(installdir string) Constants {
	return map[string]string{
		`INSTALLDIR`: installdir,
	}
}
