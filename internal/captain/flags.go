package captain

import (
	"fmt"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/rtutils/p"
)

// NameVersionFlag represents a flag that supports both a name and a version, the following formats are supported:
// - name
// - name@version
type NameVersionFlag struct {
	name    string
	version string
}

var _ FlagMarshaler = &NameVersionFlag{}

func (nv *NameVersionFlag) Set(arg string) error {
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

func (nv *NameVersionFlag) String() string {
	if nv.version == "" {
		return nv.name
	}
	return fmt.Sprintf("%s@%s", nv.name, nv.version)
}

func (nv *NameVersionFlag) Name() string {
	return nv.name
}

func (nv *NameVersionFlag) Version() string {
	return nv.version
}

func (nv *NameVersionFlag) Type() string {
	return "NameVersion"
}

// UserFlag represents a flag that supports both a name and an email address, the following formats are supported:
// - name <email>
// - email
// Emails are detected simply by containing a @ symbol.
type UserFlag struct {
	Name  string
	Email string
}

var _ FlagMarshaler = &UserFlag{}

func (u *UserFlag) String() string {
	switch {
	case u.Name == "" && u.Email == "":
		return ""
	case u.Name != "" && u.Email != "":
		return fmt.Sprintf("%s <%s>", u.Name, u.Email)
	case u.Name == "":
		return fmt.Sprintf("<%s>", u.Email)
	case u.Email == "":
		return u.Name
	}
	return ""
}

func (u *UserFlag) Set(s string) error {
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

	return locale.NewInputError("userflag_format", "Invalid format: Should be 'name <email>' or '<email>'")
}

func (u *UserFlag) Type() string {
	return "User"
}

// UsersFlag is used to represent multiple UserFlag, this is used when a flag can be passed multiple times.
type UsersFlag []UserFlag

var _ FlagMarshaler = &UsersFlag{}

func (u *UsersFlag) String() string {
	var result []string
	for _, user := range *u {
		result = append(result, user.String())
	}
	return strings.Join(result, ", ")
}

func (u *UsersFlag) Set(s string) error {
	uf := &UserFlag{}
	if err := uf.Set(s); err != nil {
		return err
	}
	*u = append(*u, *uf)
	return nil
}

func (u *UsersFlag) Type() string {
	return "Users"
}

// PackageFlag represents a flag that supports specifying a package in the following formats:
// - <namespace>/<name>
// - <namespace>/<name>@<version>
type PackageFlag struct {
	Namespace string
	Name      string
	Version   string
}

var _ FlagMarshaler = &PackageFlag{}

func (p *PackageFlag) String() string {
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

func (p *PackageFlag) Set(s string) error {
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

func (p *PackageFlag) Type() string {
	return "Package"
}

// PackageFlagNoVersion is identical to PackageFlag except that it does not support a version.
type PackageFlagNoVersion struct {
	PackageFlag
}

func (p *PackageFlagNoVersion) Set(s string) error {
	if err := p.PackageFlag.Set(s); err != nil {
		return errs.Wrap(err, "PackageFlag.Set failed")
	}
	if p.Version != "" {
		return fmt.Errorf("Specifying a version is not supported, package format should be '[<namespace>/]<name>'", s)
	}
	return nil
}

func (p *PackageFlagNoVersion) Type() string {
	return "PackageFlagNoVersion"
}

// PackageFlagNSRequired is identical to PackageFlag except that specifying a namespace is required.
type PackageFlagNSRequired struct {
	PackageFlag
}

func (p *PackageFlagNSRequired) Set(s string) error {
	if err := p.PackageFlag.Set(s); err != nil {
		return errs.Wrap(err, "PackageFlag.Set failed")
	}
	if p.Namespace == "" {
		return fmt.Errorf("invalid package name format: %s (expected '<namespace>/<name>[@version]')", s)
	}
	return nil
}

func (p *PackageFlagNSRequired) Type() string {
	return "PackageFlagNSRequired"
}

// PackagesFlag is used to represent multiple PackageFlag, this is used when a flag can be passed multiple times.
type PackagesFlag []PackageFlagNSRequired

var _ FlagMarshaler = &PackagesFlag{}

func (p *PackagesFlag) String() string {
	var result []string
	for _, pkg := range *p {
		result = append(result, pkg.String())
	}
	return strings.Join(result, ", ")
}

func (p *PackagesFlag) Set(s string) error {
	pf := &PackageFlagNSRequired{}
	if err := pf.Set(s); err != nil {
		return err
	}
	*p = append(*p, *pf)
	return nil
}

func (p *PackagesFlag) Type() string {
	return "Packages"
}

type TimeFlag struct {
	raw  string
	Time *time.Time
}

var _ FlagMarshaler = &TimeFlag{}

func (u *TimeFlag) String() string {
	return u.raw
}

func (u *TimeFlag) Set(v string) error {
	if v == "now" {
		u.Time = p.Pointer(time.Now())
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

func (u *TimeFlag) Type() string {
	return "TimeFlag"
}
