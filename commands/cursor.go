package commands

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/ambientsound/pms/input/lexer"
)

// Cursor moves the cursor in a songlist widget. It can take human-readable
// parameters such as 'up' and 'down', and it also accepts relative positions
// if a number is given.
type Cursor struct {
	api      API
	relative int
	absolute int
	current  bool
	finished bool
}

func NewCursor(api API) Command {
	return &Cursor{
		api: api,
	}
}

func (cmd *Cursor) Execute(t lexer.Token) error {
	var err error

	s := t.String()
	songlistWidget := cmd.api.SonglistWidget()

	if cmd.finished && t.Class != lexer.TokenEnd {
		return fmt.Errorf("Unknown input '%s', expected END", s)
	}

	switch t.Class {

	case lexer.TokenIdentifier:
		switch s {
		case "up":
			cmd.relative = -1
		case "down":
			cmd.relative = 1
		case "pgup":
			fallthrough
		case "pageup":
			_, y := songlistWidget.Size()
			cmd.relative = -y
		case "pgdn":
			fallthrough
		case "pagedn":
			fallthrough
		case "pagedown":
			_, y := songlistWidget.Size()
			cmd.relative = y
		case "home":
			cmd.absolute = 0
		case "end":
			cmd.absolute = songlistWidget.Len() - 1
		case "current":
			cmd.current = true
		case "random":
			len := songlistWidget.Len()
			if len > 0 {
				seed := time.Now().UnixNano()
				r := rand.New(rand.NewSource(seed))
				cmd.absolute = r.Int() % songlistWidget.Len()
			}
		default:
			i, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("Cannot move cursor: input '%s' is not recognized, and is not a number", s)
			}
			cmd.relative = i
		}
		cmd.finished = true

	case lexer.TokenEnd:
		switch {
		case !cmd.finished:
			return fmt.Errorf("Unexpected END, expected cursor offset. Try one of: up, down, pgup, pgdn, home, end, <number>")

		case cmd.current:
			currentSong := cmd.api.Song()
			if currentSong == nil {
				return fmt.Errorf("No song is currently playing.")
			}
			err = songlistWidget.CursorToSong(currentSong)

		case cmd.relative != 0:
			songlistWidget.MoveCursor(cmd.relative)

		default:
			songlistWidget.SetCursor(cmd.absolute)
		}

	default:
		return fmt.Errorf("Unknown input '%s', expected END", s)
	}

	return err
}