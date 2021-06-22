package clicommon

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

// CliQuestionYesNo prints a question prompt and allows either yes or no answers
// to be entered, returning "y" or "yes" as true and "n" or "no" as false
func CliQuestionYesNo(question string) bool {
	for {
		answer := CliQuestion(question + " (y/n)")

		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("Please only enter 'y' or 'n'")
		}
	}
}

// CliQuestionYesNoDefault prints a question prompt and allows either yes or no
// answers to be entered, or no answer at all. If no answer is entered, the
// defaultValue is returned, otherwise "y" or "yes" returns true and "n" or "no"
// returns false
func CliQuestionYesNoDefault(question string, defaultValue bool) bool {
	var yn string

	if defaultValue {
		yn = " (Y/n)"
	} else {
		yn = " (y/N)"
	}

	for {
		answer := CliQuestion(question + yn)

		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "":
			return defaultValue
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("Please enter 'y', 'n', or leave the line empty")
		}
	}
}

// CliQuestion prints a prompt to stdout and reads a line of input from stdin,
// returning that read string
func CliQuestion(question string) string {
	var answer string

	fmt.Printf("%s: ", question)
	fmt.Scanf("%s", &answer)

	return answer
}

// CliQuestionHidden prints a prompt to stdout and reads a hidden line of input
// from stdin. This is meant to be used for passwords where you don't want them
// printed to the screen
func CliQuestionHidden(question string) (string, error) {
	fmt.Printf("%s (hidden): ", question)

	rawAnswer, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}

	return string(rawAnswer), nil
}

// CliChoice provides an interactive UI to select between one or more choices
func CliChoice(question string, choices []string) (int, error) {
	if len(choices) == 0 {
		return 0, errors.New("no CLI choices provided")
	}

	// This is not a fmt.Println() because we want to use newlines intelligently
	// when printing out the choices
	fmt.Print(question)

	// Create a chooser instance
	chooser, err := newChooser(choices)
	if err != nil {
		return -1, err
	}
	defer chooser.Finish()

	// And call it
	err = chooser.Run()
	if err != nil {
		return -1, err
	}

	return chooser.index, nil
}

type chooser struct {
	origState    *term.State
	choices      []string
	index        int
	lastSearch   time.Time
	searchPrefix string
}

func newChooser(choices []string) (*chooser, error) {
	// Set the terminal into raw mode and store the original state for later
	origState, err := term.MakeRaw(0)
	if err != nil {
		return nil, err
	}

	// Hide the cursor (if supported)
	hideCursor()

	// Print the initial choices
	for i, choice := range choices {
		fmt.Println()

		if i == 0 {
			highlight()
		}

		truncated, err := truncateChoice(choice)
		if err != nil {
			return nil, err
		}

		startOfLine()
		cursorRight(4)
		fmt.Print(truncated)

		if i == 0 {
			reset()
		}
	}

	// Reset the cursor to the first choice
	prevLine(len(choices) - 1)

	return &chooser{
		origState: origState,
		choices:   choices,
	}, nil
}

func (c *chooser) Finish() error {
	// Restore the terminal state after this function exits
	defer term.Restore(0, c.origState)

	// Restore the cursor after this function exits
	defer showCursor()

	truncated, err := truncateChoice(c.choices[c.index])
	if err != nil {
		return err
	}

	// Go to the start of the first line
	if c.index > 0 {
		prevLine(c.index)
	} else {
		startOfLine()
	}

	// Erase the rest of the screen, print the selected option, and reset the
	// cursor to the start of the line for whatever comes next
	eraseRemaining()
	cursorRight(4)
	fmt.Println(truncated)
	startOfLine()

	return nil
}

func (c *chooser) Run() error {
	// Each keystroke should be a different .Read() call, so we only need enough
	// bytes to store the largest input escape sequence which is 8 bytes(?).
	// Either way, 16 bytes should be more than plenty
	buf := make([]byte, 16)

	// Process inputs and escape sequences
LOOP:
	for {
		bytesRead, err := os.Stdin.Read(buf)
		if err != nil {
			return err
		}

		if bytesRead == 0 {
			return errors.New("stdin closed")
		}

		asStr := string(buf[:bytesRead]) // note this may not be valid UTF-8

		if c.searchPrefix != "" {
			if asStr == "\x1b" {
				// If the user was searching, Escape will clear it
				c.lastSearch = time.Time{}
				c.searchPrefix = ""
				c.printChoice(0)
				continue
			}

			now := time.Now()

			if now.Sub(c.lastSearch) > time.Second {
				// If the user was last searching >1 second ago, clear it and
				// handle the current keystroke
				c.lastSearch = time.Time{}
				c.searchPrefix = ""
				c.printChoice(0)
			}
		}

		if buf[0] != '\x1b' && buf[0] > 20 && buf[0] < 127 {
			// Not an escape sequence, and a "normal" character, so search
			// with it

			c.searchPrefix += strings.ToLower(string(buf[0]))
			c.lastSearch = time.Now()
			newIndex := -1

			for i, choice := range c.choices {
				choice = strings.ToLower(choice)

				if strings.HasPrefix(choice, c.searchPrefix) {
					newIndex = i
					break
				}
			}

			if newIndex >= 0 {
				if newIndex != c.index {
					// New index, go to it
					err = c.goToIndex(newIndex)
					if err != nil {
						return err
					}
				} else {
					// No new index, so just update the current line with
					// the correct number of highlighted characters
					err = c.printChoice(len(c.searchPrefix))
					if err != nil {
						return err
					}
				}
			} else {
				err = c.printChoice(0)
				if err != nil {
					return err
				}
			}
		} else {
			// Something else, stop the search
			c.searchPrefix = ""
			c.lastSearch = time.Time{}
		}

		switch asStr {
		case "\r":
			// Plain enter to select an option
			break LOOP

		case "\x03", "\x1b":
			// Ctrl+C or Escape to cancel (unless searching, see above)
			return errors.New("operation cancelled")

		case "\x1b[A":
			// Up arrow
			err := c.up()
			if err != nil {
				return err
			}

		case "\x1b[B":
			// Down arrow
			err := c.down()
			if err != nil {
				return err
			}

		case "\x1b[1~", "\x1b[7~", "\x1b[H":
			// Home
			err := c.home()
			if err != nil {
				return err
			}

		case "\x1b[4~", "\x1b[8~", "\x1b[F":
			// End
			err := c.end()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *chooser) printChoice(highlighted int) error {
	truncated, err := truncateChoice(c.choices[c.index])
	if err != nil {
		return err
	}

	startOfLine()
	eraseLine()
	cursorRight(4)

	if highlighted == -1 {
		// -1 = off
		fmt.Print(truncated)
	} else if highlighted == 0 || highlighted == len(c.choices[c.index]) {
		// 0 = all on
		// <len> = all on
		highlight()
		fmt.Print(truncated)
		reset()
	} else {
		// >0 = number of chars on
		highlight()
		fmt.Print(truncated[:highlighted])
		reset()
		fmt.Print(truncated[highlighted:])
	}

	return nil
}

func (c *chooser) goToIndex(newIndex int) error {
	if newIndex != c.index {
		// Print the old choice without highlighting
		err := c.printChoice(-1)
		if err != nil {
			return err
		}

		// Move the cursor to the new index line
		if newIndex > c.index {
			nextLine(newIndex - c.index)
		} else {
			prevLine(c.index - newIndex)
		}

		// Save the new index number
		c.index = newIndex

		// Print the new choice with highlighting
		err = c.printChoice(len(c.searchPrefix))
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *chooser) up() error {
	if c.index > 0 {
		err := c.goToIndex(c.index - 1)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *chooser) down() error {
	if c.index < len(c.choices)-1 {
		err := c.goToIndex(c.index + 1)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *chooser) home() error {
	return c.goToIndex(0)
}

func (c *chooser) end() error {
	return c.goToIndex(len(c.choices) - 1)
}

func startOfLine() {
	fmt.Print("\x1b[G")
}

func cursorRight(chars int) {
	fmt.Printf("\x1b[%dC", chars)
}

func nextLine(lines int) {
	fmt.Printf("\x1b[%dE", lines)
}

func prevLine(lines int) {
	fmt.Printf("\x1b[%dF", lines)
}

func highlight() {
	fmt.Print("\x1b[7m")
}

func reset() {
	fmt.Print("\x1b[0m")
}

func hideCursor() {
	fmt.Print("\x1b[?25l")
}

func showCursor() {
	fmt.Print("\x1b[?25h")
}

func eraseLine() {
	fmt.Print("\x1b[2K")
}

func eraseRemaining() {
	fmt.Print("\x1b[J")
}

func truncateChoice(s string) (string, error) {
	width, _, err := term.GetSize(0)
	if err != nil {
		return "", err
	}

	maxLength := width - 4

	if len(s) > maxLength {
		return s[:maxLength-3] + "...", nil
	}

	return s, nil
}
