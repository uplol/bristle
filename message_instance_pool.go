package bristle

import (
	"sync"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// Similar to a sync.Pool but is more strictly sized
type MessageInstancePool struct {
	sync.Mutex

	wakeup *sync.Cond
	pool   []protoreflect.Message
}

func NewMessageInstancePool(messageType protoreflect.MessageType, size int) *MessageInstancePool {
	pool := make([]protoreflect.Message, size)
	for i := 0; i < size; i++ {
		pool[i] = messageType.New()
	}

	if size < 1 {
		panic("MessageInstancePool cannot have size less than 1")
	}

	p := &MessageInstancePool{
		pool: pool,
	}
	p.wakeup = sync.NewCond(p)
	return p
}

// This function has some important concurrency invariants so its thorougly documented
func (m *MessageInstancePool) Get() protoreflect.Message {
	// We acquire the pool lock to search for an existing instance that we can
	//  utilize and return to the user.
	m.Lock()

	// This variable is used to indicate the position of our selected instance
	//  within the pool. If this is -1 it indicates that we've not yet found a
	//  existing instance.
	var selectedIndex int = -1

	// We'll keep trying until we can get one
	for selectedIndex == -1 {
		// Search for an existing instance (e.g. an instance slot that is not null)
		for idx, instance := range m.pool {
			if instance != nil {
				selectedIndex = idx
				break
			}
		}

		// We've found a valid instance, we can now break and return it
		if selectedIndex != -1 {
			break
		}

		// Otherwise we need to wait until a instance is returned to the pool via
		//  the wakeup sync.Cond

		// We call Wait on the condition which **releases** our underlying embedded
		//  sync.Mutex, and blocks until the condition is hit.
		m.wakeup.Wait()

		// Coming out of this wakeup we yet again have the sync.Mutex acquired,
		//  so repeating the loop is safe.
		continue
	}

	instance := m.pool[selectedIndex]
	m.pool[selectedIndex] = nil
	m.Unlock()
	return instance
}

func (m *MessageInstancePool) Release(instance protoreflect.Message) {
	m.Lock()
	defer m.Unlock()

	var selectedIndex int = -1
	for idx, instance := range m.pool {
		if instance == nil {
			selectedIndex = idx
			break
		}
	}

	if selectedIndex == -1 {
		panic("invariant error: MessageInstancePool is full upon Release")
	}

	m.pool[selectedIndex] = instance
	m.wakeup.Signal()
}
