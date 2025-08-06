// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// ObjectID represents a object
type ObjectID interface {
	String() string
	IsZero() bool
	RawValue() []byte
	Type() ObjectFormat
}

type Sha1Hash [20]byte

// String имплементация интерфейса string
func (h *Sha1Hash) String() string {
	return hex.EncodeToString(h[:])
}

// IsZero возвращает zero value
func (h *Sha1Hash) IsZero() bool {
	empty := Sha1Hash{}
	return bytes.Equal(empty[:], h[:])
}

// RawValue создает новый срез
func (h *Sha1Hash) RawValue() []byte { return h[:] }

// Type возвращает формат объекта
func (*Sha1Hash) Type() ObjectFormat { return Sha1ObjectFormat }

var _ ObjectID = &Sha1Hash{}

type Sha256Hash [32]byte

// RawValue создает новый срез
func (h *Sha256Hash) RawValue() []byte { return h[:] }

// Type возвращает формат объекта
func (*Sha256Hash) Type() ObjectFormat { return Sha256ObjectFormat }

// String имплементация интерфейса string
func (h *Sha256Hash) String() string {
	return hex.EncodeToString(h[:])
}

// IsZero возвращает zero value
func (h *Sha256Hash) IsZero() bool {
	empty := Sha256Hash{}
	return bytes.Equal(empty[:], h[:])
}

// IsEmptyCommitID проверка на пустй коммит
func IsEmptyCommitID(commitID string) bool {
	if commitID == "" {
		return true
	}

	id, err := NewIDFromString(commitID)
	if err != nil {
		return false
	}

	return id.IsZero()
}

// ErrInvalidSHA представляет ошибку неправильного формата SHA
type ErrInvalidSHA struct {
	SHA string
}

// Error имплементация интерфейса error
func (err ErrInvalidSHA) Error() string {
	return fmt.Sprintf("invalid sha: %s", err.SHA)
}
