package termecho

func Off() error {
	return toggle(false)
}

func On() error {
	return toggle(true)
}
