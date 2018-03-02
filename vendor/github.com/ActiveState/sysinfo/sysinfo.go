package sysinfo

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
	// UnknownLibc represents an unknown C library.
	UnknownLibc
)

func (i LibcNameInfo) String() string {
	switch i {
	case Glibc:
		return "glibc"
	case Msvcrt:
		return "msvcrt"
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
	// Clang represents the LLVM/Clang toolchain.
	Clang
	// Msvc represents the Microsoft Visual C++ toolchain.
	Msvc
	// Mingw represents the Minimalist GNU for Windows toolchain.
	Mingw
	// Cygwin represents the Cygwin toolchain.
	Cygwin
)

func (i CompilerNameInfo) String() string {
	switch i {
	case Gcc:
		return "GCC"
	case Clang:
		return "clang"
	case Msvc:
		return "MSVC"
	case Mingw:
		return "MinGW"
	case Cygwin:
		return "Cygwin"
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
