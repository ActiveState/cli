package ct

import (
	"fmt"
	"os"
	"testing"
)

func TestChangeColor(t *testing.T) {
	defer Reset(os.Stdout)
	fmt.Println("Normal text...")
	text := "This is an demo of using ChangeColor to output colorful texts"
	i := 1
	for _, c := range text {
		ChangeColor(os.Stdout, Color(i/2%8)+Black, i%2 == 1, Color((i+2)/2%8)+Black, false)
		fmt.Print(string(c))
		i++
	}
	fmt.Println()
	ChangeColor(os.Stdout, Red, true, White, false)
	fmt.Println("Before reset.")
	ChangeColor(os.Stdout, Red, false, White, true)
	fmt.Println("Before reset.")
	Reset(os.Stdout)
	fmt.Println("After reset.")
	fmt.Println("After reset.")
}

func TestChangeStyle(t *testing.T) {
	defer Reset(os.Stdout)
	fmt.Println("Normal text...")
	fmt.Println()

	ChangeStyle(os.Stdout, Bold)
	fmt.Println("Bold, not underlined")
	Reset(os.Stdout)
	fmt.Println("Reset")

	ChangeStyle(os.Stdout, Underline)
	fmt.Println("Underlined, not bold")
	Reset(os.Stdout)
	fmt.Println("Reset")

	ChangeStyle(os.Stdout, Underline, Bold)
	fmt.Println("Underlined, Bold")
	Reset(os.Stdout)
	fmt.Println("Reset")
}

func TestForeground(t *testing.T) {
	Reset(os.Stdout)
	defer Reset(os.Stdout)

	fmt.Println("Please check the words under the following text shows with the corresponding front color:")

	colorToText := [...]string{
		Black:   "black",
		Red:     "red",
		Green:   "green",
		Yellow:  "yellow",
		Blue:    "blue",
		Magenta: "magenta",
		Cyan:    "cyan",
		White:   "white",
	}
	for i, txt := range colorToText {
		cl := Color(i)
		if cl != None {
			Foreground(os.Stdout, cl, false)
			fmt.Print(txt, ",")
			Foreground(os.Stdout, cl, true)
			fmt.Print(txt, ",")
		}
	}
	fmt.Println()
}

func TestBackground(t *testing.T) {
	Reset(os.Stdout)
	defer Reset(os.Stdout)

	fmt.Println("Please check the words under the following text shows with the corresponding background color:")

	colorToText := [...]string{
		Black:   "black",
		Red:     "red",
		Green:   "green",
		Yellow:  "yellow",
		Blue:    "blue",
		Magenta: "magenta",
		Cyan:    "cyan",
		White:   "white",
	}
	for i, txt := range colorToText {
		cl := Color(i)
		if cl != None {
			Background(os.Stdout, cl, false)
			fmt.Print(txt, ",")
			Background(os.Stdout, cl, true)
			fmt.Print(txt, ",")
		}
	}
	fmt.Println()
}
