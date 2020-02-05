package platforms

// Printer describes what is needed for platforms to print info.
type Printer interface {
	Print(interface{})
}
