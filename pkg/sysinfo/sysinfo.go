package sysinfo

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/patrickmn/go-cache"
)

var sysinfoCache *cache.Cache = cache.New(cache.NoExpiration, cache.NoExpiration)

const VersionOverrideEnvVar = "ACTIVESTATE_CLI_OSVERSION_OVERRIDE"

// Cache keys used for storing/retrieving computed system information.
const (
	osVersionInfoCacheKey = "osVersionInfo"
	libcInfoCacheKey      = "libcInfo"
	compilersCacheKey     = "compilers"
)

// OsInfo represents an OS returned by OS().
type OsInfo int

const (
	// Linux represents the Linux operating system.
	Linux OsInfo = iota
	// Windows represents the Windows operating system.
	Windows
	// Mac represents the Macintosh operating system.
	Mac
	// UnknownOs represents an unknown operating system.
	UnknownOs
)

func (i OsInfo) String() string {
	switch i {
	case Linux:
		return "Linux"
	case Windows:
		return "Windows"
	case Mac:
		return "MacOS"
	default:
		return "Unknown"
	}
}

// OSVersionInfo represents an OS version returned by OSVersion().
type OSVersionInfo struct {
	Version string // raw version string
	Major   int    // major version number
	Minor   int    // minor version number
	Micro   int    // micro version number
	Name    string // free-form name string (varies by OS)
}

// ArchInfo represents an architecture returned by Architecture().
type ArchInfo int

const (
	// I386 represents the Intel x86 (32-bit) architecture.
	I386 ArchInfo = iota
	// Amd64 represents the x86_64 (64-bit) architecture.
	Amd64
	// Arm represents the ARM architecture.
	Arm
	// UnknownArch represents an unknown architecture.
	UnknownArch
)

func (i ArchInfo) String() string {
	switch i {
	case I386:
		return "i386"
	case Amd64:
		return "x86_64"
	case Arm:
		return "ARM"
	default:
		return "Unknown"
	}
}

// LibcNameInfo represents a C library name.
type LibcNameInfo int

const (
	// Glibc represents the GNU C library.
	Glibc LibcNameInfo = iota
	// Msvcrt represents the Microsoft Visual C++ runtime library.
	Msvcrt
	// BsdLibc represents the BSD C library.
	BsdLibc
	// UnknownLibc represents an unknown C library.
	UnknownLibc
)

func (i LibcNameInfo) String() string {
	switch i {
	case Glibc:
		return "glibc"
	case Msvcrt:
		return "msvcrt"
	case BsdLibc:
		return "libc"
	default:
		return "Unknown"
	}
}

// LibcInfo represents a LibC returned by Libc().
type LibcInfo struct {
	Name  LibcNameInfo // C library name
	Major int          // major version number
	Minor int          // minor version number
}

// CompilerNameInfo reprents a compiler toolchain name.
type CompilerNameInfo int

const (
	// Gcc represents the GNU C Compiler toolchain.
	Gcc CompilerNameInfo = iota
	// Msvc represents the Microsoft Visual C++ toolchain.
	Msvc
	// Mingw represents the Minimalist GNU for Windows toolchain.
	Mingw
	// Clang represents the LLVM/Clang toolchain.
	Clang
)

func (i CompilerNameInfo) String() string {
	switch i {
	case Gcc:
		return "GCC"
	case Msvc:
		return "MSVC"
	case Mingw:
		return "MinGW"
	case Clang:
		return "clang"
	default:
		return "Unknown"
	}
}

// CompilerInfo represents a compiler toolchain returned by Compiler().
type CompilerInfo struct {
	Name  CompilerNameInfo // C compiler name
	Major int              // major version number
	Minor int              // minor version number
}

// Checks whether or not the given compiler exists and returns its major and
// minor version numbers. A major return of 0 indicates the compiler does not
// exist, or that an error occurred.
func getCompilerVersion(args []string) (int, int, error) {
	cc, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return 0, 0, nil
	}
	regex := regexp.MustCompile("(\\d+)\\D(\\d+)\\D\\d+")
	parts := regex.FindStringSubmatch(string(cc))
	if len(parts) != 3 {
		return 0, 0, fmt.Errorf("Unable to parse version string '%s'", cc)
	}
	for i := 1; i < len(parts); i++ {
		if _, err := strconv.Atoi(parts[i]); err != nil {
			return 0, 0, fmt.Errorf("Unable to parse part '%s' of version string '%s'", parts[i], cc)
		}
	}
	major, _ := strconv.Atoi(parts[1])
	minor, _ := strconv.Atoi(parts[2])
	return major, minor, nil
}
