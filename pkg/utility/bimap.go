package pkgUtility

type Bimap[K comparable, V comparable] struct {
	keyToValue map[K]V
	valueToKey map[V]K
}

func NewBimap[K comparable, V comparable]() *Bimap[K, V] {
	return &Bimap[K, V]{
		keyToValue: make(map[K]V),
		valueToKey: make(map[V]K),
	}
}

func (m *Bimap[K, V]) Set(key K, value V) {
	m.keyToValue[key] = value
	m.valueToKey[value] = key
}

func (m *Bimap[K, V]) Get(key K) (V, bool) {
	value, ok := m.keyToValue[key]
	return value, ok
}

func (m *Bimap[K, V]) GetKey(value V) (K, bool) {
	key, ok := m.valueToKey[value]
	return key, ok
}

func (m *Bimap[K, V]) Delete(key K) {
	value, ok := m.keyToValue[key]
	if !ok {
		return
	}

	delete(m.keyToValue, key)
	delete(m.valueToKey, value)
}
