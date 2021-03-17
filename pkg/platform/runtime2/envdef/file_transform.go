package envdef

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

// FileTransform specifies a single transformation to be performed on files in artifacts post-installation
type FileTransform struct {
	Pattern         string           `json:"pattern"`
	In              []string         `json:"in"`
	With            string           `json:"with"`
	ConstTransforms []ConstTransform `json:"const_transforms"`
	PadWith         *string          `json:"pad_with"`
}

// ConstTransform is a transformation that should be applied to substituted constants prior to substitution in files
type ConstTransform struct {
	In      []string `json:"in"` // List of constants to apply this transform to
	Pattern string   `json:"pattern"`
	With    string   `json:"with"`
}

// applyConstTransforms applies the constant transforms to the Constants values
func (ft *FileTransform) applyConstTransforms(constants Constants) (Constants, error) {
	// copy constants, such that we don't change it
	cs := make(Constants)
	for k, v := range constants {
		cs[k] = v
	}
	for _, ct := range ft.ConstTransforms {
		for _, inVar := range ct.In {
			inSubst, ok := cs[inVar]
			if !ok {
				return cs, errs.New("Do not know what to replace constant %s with.", inVar)
			}
			cs[inVar] = strings.ReplaceAll(inSubst, string(ct.Pattern), string(ct.With))
		}
	}

	return cs, nil
}

func (ft *FileTransform) relocateFile(fileBytes []byte, replacement string) ([]byte, error) {
	findBytes := []byte(ft.Pattern)
	replacementBytes := []byte(replacement)

	// If `pad_width == null`, no padding is necessary and we can just replace the string and return
	if ft.PadWith == nil {
		return bytes.ReplaceAll(fileBytes, findBytes, replacementBytes), nil
	}

	// padding should be one byte
	if len(*ft.PadWith) != 1 {
		return fileBytes, errs.New("Padding character needs to have exactly one byte, got %d", len(*ft.PadWith))
	}
	pad := []byte(*ft.PadWith)[0]

	// replacement should be shorter than search string
	if len(replacementBytes) > len(findBytes) {
		logging.Errorf("Replacement text too long: %s, original text: %s", ft.Pattern, replacement)
		return fileBytes, locale.NewError("file_transform_replacement_too_long", "Replacement text cannot be longer than search text in a binary file.")
	}

	// Must account for the expand characters (ie. '${1}') in the
	// replacement bytes in order for the binary paddding to be correct
	regexExpandBytes := []byte("${1}")
	replacementBytes = append(replacementBytes, regexExpandBytes...)

	// paddedReplaceBytes is the replacement string plus the padding bytes added to the end
	// It shall look like this: `<replacementBytes>${1}<padding>` with `len(replacementBytes)+len(padding)=len(findBytes)`
	paddedReplaceBytes := bytes.Repeat([]byte{pad}, len(findBytes)+len(regexExpandBytes))
	copy(paddedReplaceBytes, replacementBytes)

	quoteEscapeFind := regexp.QuoteMeta(ft.Pattern)
	// replacementRegex matches the search Pattern plus subsequent text up to the string termination character (pad, which usually is 0x00)
	replacementRegex, err := regexp.Compile(fmt.Sprintf(`%s([^\x%02x]*)`, quoteEscapeFind, pad))
	if err != nil {
		return fileBytes, errs.Wrap(err, "Failed to compile replacement regular expression.")
	}
	return replacementRegex.ReplaceAll(fileBytes, paddedReplaceBytes), nil
}

func expandConstants(in string, constants Constants) string {
	res := in
	for k, v := range constants {
		res = strings.ReplaceAll(res, fmt.Sprintf("${%s}", k), v)
	}
	return res
}

// ApplyTransform applies a file transformation to all specified files
func (ft *FileTransform) ApplyTransform(baseDir string, constants Constants) error {
	// compute transformed constants
	tcs, err := ft.applyConstTransforms(constants)
	if err != nil {
		return errs.Wrap(err, "Failed to apply the constant transformation to replacement text.")
	}
	replacement := expandConstants(ft.With, tcs)

	for _, f := range ft.In {
		fp := filepath.Join(baseDir, f)
		fileBytes, err := ioutil.ReadFile(fp)
		if err != nil {
			return errs.Wrap(err, "Could not read file contents of %s.", fp)
		}

		replaced, err := ft.relocateFile(fileBytes, replacement)
		if err != nil {
			return errs.Wrap(err, "relocateFile failed")
		}

		// skip writing back to file if contents remain the same after transformation
		if bytes.Equal(replaced, fileBytes) {
			continue
		}

		err = fileutils.WriteFile(fp, replaced)
		if err != nil {
			return errs.Wrap(err, "Could not write file contents.")
		}
	}

	return nil
}

// ApplyFileTransforms applies all file transformations to the files in the base directory
func (ed *EnvironmentDefinition) ApplyFileTransforms(installDir string, constants Constants) error {
	for _, ft := range ed.Transforms {
		err := ft.ApplyTransform(installDir, constants)
		if err != nil {
			return err
		}
	}
	return nil
}
