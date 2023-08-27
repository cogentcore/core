package mobile

import (
	"sync"

	"goki.dev/gi/oswin/cursor"
)

/////////////////////////////////////////////////////////////////
// cursor impl (note: cursor does not exist on mobile, so the functions do nothing)

type cursorImpl struct {
	cursor.CursorBase
	mu sync.Mutex
}

var theCursor = cursorImpl{CursorBase: cursor.CursorBase{Vis: false}}

func (c *cursorImpl) Push(sh cursor.Shapes) {}

func (c *cursorImpl) Set(sh cursor.Shapes) {}

func (c *cursorImpl) Pop() {}

func (c *cursorImpl) Hide() {}

func (c *cursorImpl) Show() {}

func (c *cursorImpl) PushIfNot(sh cursor.Shapes) bool { return false }

func (c *cursorImpl) PopIf(sh cursor.Shapes) bool { return false }
