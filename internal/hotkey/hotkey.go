package hotkey

var currentHandler Handler

func Register(hotkey string, handler Handler) error {
	currentHandler = handler
	return register(hotkey)
}

func Unregister() {
	unregister()
}

func ChangeHotkey(newHotkey string, handler Handler) error {
	Unregister()
	currentHandler = handler
	return register(newHotkey)
}
