package captain

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
)

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
// - name
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
		return nil
	}

	u.Name = strings.TrimSpace(s)
	return nil
}

func (u *UserFlag) Type() string {
	return "User"
}

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

// PackageFlag represents a flag that supports specifying a package in the following format:
// <namsepace>/<name>@<version>
type PackageFlag struct {
	Namespace string
	Name      string
	Version   string
}

var _ FlagMarshaler = &PackageFlag{}

func (u *PackageFlag) String() string {
	if u.Namespace == "" && u.Name == "" {
		return ""
	}
	if u.Version == "" {
		return fmt.Sprintf("%s/%s", u.Namespace, u.Name)
	}
	return fmt.Sprintf("%s/%s@%s", u.Namespace, u.Name, u.Version)
}

func (u *PackageFlag) Set(s string) error {
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

func (u *PackageFlag) Type() string {
	return "Package"
}

type PackagesFlag []PackageFlag

var _ FlagMarshaler = &PackagesFlag{}

func (p *PackagesFlag) String() string {
	var result []string
	for _, pkg := range *p {
		result = append(result, pkg.String())
	}
	return strings.Join(result, ", ")
}

func (p *PackagesFlag) Set(s string) error {
	pf := &PackageFlag{}
	if err := pf.Set(s); err != nil {
		return err
	}
	*p = append(*p, *pf)
	return nil
}

func (p *PackagesFlag) Type() string {
	return "Packages"
}
