package bristle

type messageBindingRegistry map[string]*MessageTableBinding

func newMessageBindingRegistry() messageBindingRegistry {
	return make(messageBindingRegistry)
}
