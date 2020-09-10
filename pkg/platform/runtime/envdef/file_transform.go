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

// applyConstTransforms applies the constant transforms to the FileTransform.With field
func (ft *FileTransform) applyConstTransforms(constants Constants) (string, error) {
	with := ft.With
	for _, ct := range ft.ConstTransforms {
		r, err := regexp.Compile(ct.Pattern)
		if err != nil {
			return "", errs.Wrap(err, "file_transform_compile_const_pattern_err", "Failed to compile regexp pattern in const_transform.")
		}
		for _, inVar := range ct.In {
			inSubst, ok := constants[inVar]
			if !ok {
				return "", errs.New("file_tranform_unknown_constant", "Do not know what to replace constant {{.V0}} with.", inVar)
			}
			tSubst := r.ReplaceAllString(inSubst, ct.With)
			with = strings.ReplaceAll(with, fmt.Sprintf("${%s}", inVar), tSubst)
		}
	}

	return with, nil
}

func (ft *FileTransform) relocateFile(fileBytes []byte, replacement string) ([]byte, error) {
	findBytes := []byte(ft.Pattern)
	replacementBytes := []byte(replacement)

	if ft.PadWith == nil {
		return bytes.ReplaceAll(fileBytes, findBytes, replacementBytes), nil
	}

	// padding should be one byte
	if len(*ft.PadWith) != 1 {
		return fileBytes, errs.New("file_transform_invalid_padding", "Padding character needs to have exactly one byte, got {{.V0}}", len(*ft.PadWith))
	}

	// replacement should be shorter than search string
	if len(replacementBytes) > len(findBytes) {
		logging.Errorf("Replacement text too long: %s, original text: %s", ft.Pattern, replacement)
		return fileBytes, errs.New("file_transform_replacement_too_long", "Replacement text cannot be longer than search text in a binary file.")
	}

	regexExpandBytes := []byte("${1}")
	// Must account for the expand characters (ie. '${1}') in the
	// replacement bytes in order for the binary paddding to be correct
	replacementBytes = append(replacementBytes, regexExpandBytes...)

	pad := []byte(*ft.PadWith)[0]
	paddedReplaceBytes := bytes.Repeat([]byte{pad}, len(findBytes)+len(regexExpandBytes))
	copy(paddedReplaceBytes, replacementBytes)

	quoteEscapeFind := regexp.QuoteMeta(ft.Pattern)
	replacementRegex, err := regexp.Compile(fmt.Sprintf(`%s([^\\x%02x]*)`, quoteEscapeFind, pad))
	if err != nil {
		return fileBytes, errs.Wrap(err, "file_transform_regexp_compile_err", "Failed to compile replacement regular expression.")
	}

	return replacementRegex.ReplaceAll(fileBytes, paddedReplaceBytes), nil
}

// ApplyTransform applies a file transformation to all specified files
func (ft *FileTransform) ApplyTransform(baseDir string, constants Constants) error {
	replacement, err := ft.applyConstTransforms(constants)
	if err != nil {
		return errs.Wrap(err, "file_transform_const_transform_err", "Failed to apply the constant transformation to replacement text.")
	}

	for _, f := range ft.In {
		fp := filepath.Join(baseDir, f)
		fileBytes, err := ioutil.ReadFile(fp)
		if err != nil {
			return errs.Wrap(err, "file_transform_read_file_err", "Could not read file contents.")
		}

		replaced, err := ft.relocateFile(fileBytes, replacement)
		if err != nil {
			return err
		}

		// skip writing back to file if contents remain the same after transformation
		if bytes.Equal(replaced, fileBytes) {
			continue
		}

		fail := fileutils.WriteFile(fp, replaced)
		if fail != nil {
			return errs.Wrap(fail.ToError(), "file_transform_write_file_err", "Could not write file contents.")
		}
	}

	return nil
}

// ApplyFileTransforms applies all file transformations to the files in the base directory
func (ed *EnvironmentDefinition) ApplyFileTransforms(baseDir string, constants Constants) error {
	for _, ft := range ed.Transforms {
		err := ft.ApplyTransform(baseDir, constants)
		if err != nil {
			return err
		}
	}
	return nil
}
