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

func fnvHash[K comparable](key K) (uint64, error) {
	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(key); err != nil {
		return 0, fmt.Errorf("failed to encode key, seems like key is nil, err: %v", err)
	}

	hasher := fnv.New64()
	if _, err := hasher.Write(b.Bytes()); err != nil {
		return 0, fmt.Errorf("failed to write key using 64-bit fnv hash, err: %v", err)
	}
	return hasher.Sum64(), nil
}

func wrapEntry[V any](hashKey uint64, value V) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	entry := buf.Bytes()

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
