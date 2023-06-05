package captain

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
)

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
	return "NameVersion"
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
	return "User"
}

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
	return "Users"
}

// PackageValue represents a flag that supports specifying a package in the following format:
// <namsepace>/<name>@<version>
type PackageValue struct {
	Namespace string
	Name      string
	Version   string
}

var _ FlagMarshaler = &PackageValue{}

func (u *PackageValue) String() string {
	if u.Namespace == "" && u.Name == "" {
		return ""
	}
	if u.Version == "" {
		return fmt.Sprintf("%s/%s", u.Namespace, u.Name)
	}
	return fmt.Sprintf("%s/%s@%s", u.Namespace, u.Name, u.Version)
}

func (u *PackageValue) Set(s string) error {
	if strings.Contains(s, "@") {
		v := strings.Split(s, "@")
		u.Version = strings.TrimSpace(v[1])
		s = v[0]
	}
	if strings.Index(s, "/") == -1 {
		return fmt.Errorf("invalid package name format: %s (expected '<namespace>/<name>[@version]')", s)
	}
	v := strings.Split(s, "/")
	u.Namespace = strings.TrimSpace(strings.Join(v[0:len(v)-1], "/"))
	u.Name = strings.TrimSpace(v[len(v)-1])
	return nil
}

func (u *PackageValue) Type() string {
	return "Package"
}

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
	return "Packages"
}
