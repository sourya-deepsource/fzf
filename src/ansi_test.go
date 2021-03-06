package fzf

import (
	"fmt"
	"strings"
	"testing"

	"github.com/junegunn/fzf/src/tui"
)

func TestExtractColor(t *testing.T) {
	assert := func(offset ansiOffset, b int32, e int32, fg tui.Color, bg tui.Color, bold bool) {
		var attr tui.Attr
		if bold {
			attr = tui.Bold
		}
		if offset.offset[0] != b || offset.offset[1] != e ||
			offset.color.fg != fg || offset.color.bg != bg || offset.color.attr != attr {
			t.Error(offset, b, e, fg, bg, attr)
		}
	}

	src := "hello world"
	var state *ansiState
	clean := "\x1b[0m"
	check := func(assertion func(ansiOffsets *[]ansiOffset, state *ansiState)) {
		output, ansiOffsets, newState := extractColor(src, state, nil)
		state = newState
		if output != "hello world" {
			t.Errorf("Invalid output: %s %v", output, []rune(output))
		}
		fmt.Println(src, ansiOffsets, clean)
		assertion(ansiOffsets, state)
	}

	check(func(offsets *[]ansiOffset, state *ansiState) {
		if offsets != nil {
			t.Fail()
		}
	})

	state = nil
	src = "\x1b[0mhello world"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if offsets != nil {
			t.Fail()
		}
	})

	state = nil
	src = "\x1b[1mhello world"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		assert((*offsets)[0], 0, 11, -1, -1, true)
	})

	state = nil
	src = "\x1b[1mhello \x1b[mw\x1b7o\x1b8r\x1b(Bl\x1b[2@d"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		assert((*offsets)[0], 0, 6, -1, -1, true)
	})

	state = nil
	src = "\x1b[1mhello \x1b[Kworld"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		assert((*offsets)[0], 0, 11, -1, -1, true)
	})

	state = nil
	src = "hello \x1b[34;45;1mworld"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		assert((*offsets)[0], 6, 11, 4, 5, true)
	})

	state = nil
	src = "hello \x1b[34;45;1mwor\x1b[34;45;1mld"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		assert((*offsets)[0], 6, 11, 4, 5, true)
	})

	state = nil
	src = "hello \x1b[34;45;1mwor\x1b[0mld"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		assert((*offsets)[0], 6, 9, 4, 5, true)
	})

	state = nil
	src = "hello \x1b[34;48;5;233;1mwo\x1b[38;5;161mr\x1b[0ml\x1b[38;5;161md"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 3 {
			t.Fail()
		}
		assert((*offsets)[0], 6, 8, 4, 233, true)
		assert((*offsets)[1], 8, 9, 161, 233, true)
		assert((*offsets)[2], 10, 11, 161, -1, false)
	})

	// {38,48};5;{38,48}
	state = nil
	src = "hello \x1b[38;5;38;48;5;48;1mwor\x1b[38;5;48;48;5;38ml\x1b[0md"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 2 {
			t.Fail()
		}
		assert((*offsets)[0], 6, 9, 38, 48, true)
		assert((*offsets)[1], 9, 10, 48, 38, true)
	})

	src = "hello \x1b[32;1mworld"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		if state.fg != 2 || state.bg != -1 || state.attr == 0 {
			t.Fail()
		}
		assert((*offsets)[0], 6, 11, 2, -1, true)
	})

	src = "hello world"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		if state.fg != 2 || state.bg != -1 || state.attr == 0 {
			t.Fail()
		}
		assert((*offsets)[0], 0, 11, 2, -1, true)
	})

	src = "hello \x1b[0;38;5;200;48;5;100mworld"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 2 {
			t.Fail()
		}
		if state.fg != 200 || state.bg != 100 || state.attr > 0 {
			t.Fail()
		}
		assert((*offsets)[0], 0, 6, 2, -1, true)
		assert((*offsets)[1], 6, 11, 200, 100, false)
	})
}

func TestAnsiCodeStringConversion(t *testing.T) {
	assert := func(code string, prevState *ansiState, expected string) {
		state := interpretCode(code, prevState)
		if expected != state.ToString() {
			t.Errorf("expected: %s, actual: %s",
				strings.ReplaceAll(expected, "\x1b[", "\\x1b["),
				strings.ReplaceAll(state.ToString(), "\x1b[", "\\x1b["))
		}
	}
	assert("\x1b[m", nil, "")
	assert("\x1b[m", &ansiState{attr: tui.Blink}, "")

	assert("\x1b[31m", nil, "\x1b[31;49m")
	assert("\x1b[41m", nil, "\x1b[39;41m")

	assert("\x1b[92m", nil, "\x1b[92;49m")
	assert("\x1b[102m", nil, "\x1b[39;102m")

	assert("\x1b[31m", &ansiState{fg: 4, bg: 4}, "\x1b[31;44m")
	assert("\x1b[1;2;31m", &ansiState{fg: 2, bg: -1, attr: tui.Reverse}, "\x1b[1;2;7;31;49m")
	assert("\x1b[38;5;100;48;5;200m", nil, "\x1b[38;5;100;48;5;200m")
	assert("\x1b[48;5;100;38;5;200m", nil, "\x1b[38;5;200;48;5;100m")
	assert("\x1b[48;5;100;38;2;10;20;30;1m", nil, "\x1b[1;38;2;10;20;30;48;5;100m")
	assert("\x1b[48;5;100;38;2;10;20;30;7m",
		&ansiState{attr: tui.Dim | tui.Italic, fg: 1, bg: 1},
		"\x1b[2;3;7;38;2;10;20;30;48;5;100m")
}
