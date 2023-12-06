package fifo

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"hash/fnv"
)

const (
	headerSizeInByte = 8 // 8 bytes for uint64
)

func fnvHash[K comparable](key K) uint64 {
	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(key); err != nil {
		panic(fmt.Sprintf("key cannot be nil err: %v", err))
	}

	hasher := fnv.New64()
	_, _ = hasher.Write(b.Bytes()) // ignore because this does not return error
	return hasher.Sum64()
}

func wrapEntry[V any](hashKey uint64, value V) ([]byte, error) {
	entry, err := encode[V](value)
	if err != nil {
		return nil, err
	}

	blobLength := len(entry) + headerSizeInByte
	blob := make([]byte, blobLength)

	binary.LittleEndian.PutUint64(blob, hashKey)
	copy(blob[headerSizeInByte:], entry)

	return blob, nil
}

func unwrapEntry[V any](blob []byte) (hashKey uint64, value V, err error) {
	hashKey = binary.LittleEndian.Uint64(blob[:headerSizeInByte])
	value, err = decode[V](blob[headerSizeInByte:])
	return
}

func encode[V any](value V) (ret []byte, err error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err = enc.Encode(value)
	if err != nil {
		return nil, err
	}
	ret = buf.Bytes()
	return
}

func decode[V any](data []byte) (value V, err error) {
	dec := gob.NewDecoder(bytes.NewReader(data))
	err = dec.Decode(&value)
	return
}
