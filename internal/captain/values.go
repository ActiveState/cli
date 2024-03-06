package captain

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// NameVersionValue represents a flag that supports both a name and a version, the following formats are supported:
// - name
// - name@version
type NameVersionValue struct {
	name    string
	version string
}

var _ FlagMarshaler = &NameVersionValue{}

func (nv *NameVersionValue) Set(arg string) error {
	nameArg := strings.Split(arg, "@")
	nv.name = nameArg[0]
	if len(nameArg) == 2 {
		nv.version = nameArg[1]
	}
	if len(nameArg) > 2 {
		return locale.NewInputError("name_version_format_err", "Invalid format: Should be <name@version>")
	}
	return nil
}

func (nv *NameVersionValue) String() string {
	if nv.version == "" {
		return nv.name
	}
	return fmt.Sprintf("%s@%s", nv.name, nv.version)
}

func (nv *NameVersionValue) Name() string {
	return nv.name
}

func (nv *NameVersionValue) Version() string {
	return nv.version
}

func (nv *NameVersionValue) Type() string {
	return "name and version"
}

// UserValue represents a flag that supports both a name and an email address, the following formats are supported:
// - name <email>
// - email
// Emails are detected simply by containing a @ symbol.
type UserValue struct {
	Name  string
	Email string
}

var _ FlagMarshaler = &UserValue{}

func (u *UserValue) String() string {
	switch {
	case u.Name != "" && u.Email != "":
		return fmt.Sprintf("%s <%s>", u.Name, u.Email)
	case u.Email != "":
		return fmt.Sprintf("<%s>", u.Email)
	}
	return u.Name
}

func (u *UserValue) Set(s string) error {
	if strings.Contains(s, "<") {
		v := strings.Split(s, "<")
		u.Name = strings.TrimSpace(v[0])
		u.Email = strings.TrimRight(strings.TrimSpace(v[1]), ">")
		return nil
	}

	if strings.Contains(s, "@") {
		u.Email = strings.TrimSpace(s)
		u.Name = strings.Split(u.Email, "@")[0]
		return nil
	}

	return locale.NewInputError("uservalue_format", "Invalid format: Should be 'name <email>' or '<email>'")
}

func (u *UserValue) Type() string {
	return "user"
}

// UsersValue is used to represent multiple UserValue, this is used when a flag can be passed multiple times.
type UsersValue []UserValue

var _ FlagMarshaler = &UsersValue{}

func (u *UsersValue) String() string {
	var result []string
	for _, user := range *u {
		result = append(result, user.String())
	}
	return strings.Join(result, ", ")
}

func (u *UsersValue) Set(s string) error {
	uf := &UserValue{}
	if err := uf.Set(s); err != nil {
		return err
	}
	*u = append(*u, *uf)
	return nil
}

func (u *UsersValue) Type() string {
	return "users"
}

// PackageValue represents a flag that supports specifying a package in the following formats:
// - <name>
// - <namespace>/<name>
// - <namespace>/<name>@<version>
type PackageValue struct {
	Namespace string
	Name      string
	Version   string
}

var _ FlagMarshaler = &PackageValue{}

func (p *PackageValue) String() string {
	if p.Namespace == "" && p.Name == "" {
		return ""
	}
	name := p.Name
	if p.Namespace != "" {
		name = fmt.Sprintf("%s/%s", p.Namespace, p.Name)
	}
	if p.Version == "" {
		return name
	}
	return fmt.Sprintf("%s@%s", name, p.Version)
}

func (p *PackageValue) Set(s string) error {
	if strings.Contains(s, "@") {
		v := strings.Split(s, "@")
		p.Version = strings.TrimSpace(v[1])
		s = v[0]
	}
	if strings.Index(s, "/") == -1 {
		p.Name = strings.TrimSpace(s)
		return nil
	}
	v := strings.Split(s, "/")
	p.Namespace = strings.TrimSpace(strings.Join(v[0:len(v)-1], "/"))
	p.Name = strings.TrimSpace(v[len(v)-1])
	return nil
}

func (p *PackageValue) Type() string {
	return "package"
}

// PackageValueNoVersion is identical to PackageValue except that it does not support a version.
type PackageValueNoVersion struct {
	PackageValue
}

func (p *PackageValueNoVersion) Set(s string) error {
	if err := p.PackageValue.Set(s); err != nil {
		return errs.Wrap(err, "PackageValue.Set failed")
	}
	if p.Version != "" {
		return fmt.Errorf("Specifying a version is not supported, package format should be '[<namespace>/]<name>'")
	}
	return nil
}

func (p *PackageValueNoVersion) Type() string {
	return "package"
}

// PackageValueNSRequired is identical to PackageValue except that specifying a namespace is required.
type PackageValueNSRequired struct {
	PackageValue
}

func (p *PackageValueNSRequired) Set(s string) error {
	if err := p.PackageValue.Set(s); err != nil {
		return errs.Wrap(err, "PackageValueNSRequired.Set failed")
	}
	if p.Namespace == "" {
		return fmt.Errorf("invalid package name format: %s (expected '<namespace>/<name>[@version]')", s)
	}
	return nil
}
func (p *PackageValueNSRequired) Type() string {
	return "namespace/package"
}

// PackagesValue is used to represent multiple PackageValue, this is used when a flag can be passed multiple times.
type PackagesValue []PackageValue

var _ FlagMarshaler = &PackagesValue{}

func (p *PackagesValue) String() string {
	var result []string
	for _, pkg := range *p {
		result = append(result, pkg.String())
	}
	return strings.Join(result, ", ")
}

func (p *PackagesValue) Set(s string) error {
	pf := &PackageValue{}
	if err := pf.Set(s); err != nil {
		return err
	}
	*p = append(*p, *pf)
	return nil
}

func (p *PackagesValue) Type() string {
	return "packages"
}

type TimeValue struct {
	raw  string
	Time *time.Time
}

var _ FlagMarshaler = &TimeValue{}

func (u *TimeValue) String() string {
	return u.raw
}

func (u *TimeValue) Set(v string) error {
	if v == "now" {
		latest, err := model.FetchLatestRevisionTimeStamp(authentication.LegacyGet())
		if err != nil {
			multilog.Error("Unable to determine latest revision time: %v", err)
			latest = time.Now()
		}
		u.Time = ptr.To(latest)
	} else {
		u.raw = v
		tsv, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return locale.WrapInputError(err, "timeflag_format", "Invalid timestamp: Should be RFC3339 formatted.")
		}
		u.Time = &tsv
	}
	return nil
}

func (u *TimeValue) Type() string {
	return "timestamp"
}

type IntValue struct {
	raw string
	Int *int
}

var _ FlagMarshaler = &IntValue{}

func (i *IntValue) String() string {
	return i.raw
}

func (i *IntValue) Set(v string) error {
	if v == "" {
		return nil
	}

	iv, err := strconv.Atoi(v)
	if err != nil {
		return locale.WrapInputError(err, "intflag_format", "Invalid int: Should be an integer, got: {{.V0}}.", v)
	}
	i.raw = v
	i.Int = &iv
	return nil
}

func (i *IntValue) Type() string {
	return "int"
}

func (i *IntValue) IsSet() bool {
	return i.Int != nil
}
