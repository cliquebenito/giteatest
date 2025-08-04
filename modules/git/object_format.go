// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"crypto/sha1"
	"crypto/sha256"
	"regexp"
	"strconv"

	"code.gitea.io/gitea/modules/log"
)

// sha1Pattern can be used to determine if a string is an valid sha
var sha1Pattern = regexp.MustCompile(`^[0-9a-f]{4,40}$`)

// sha256Pattern can be used to determine if a string is an valid sha
var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{4,64}$`)

// ObjectFormat interface represents a object formats
type ObjectFormat interface {
	// Name returns the name of the object format
	Name() string
	// EmptyObjectID creates a new empty ObjectID from an object format hash name
	EmptyObjectID() ObjectID
	// EmptyTree is the hash of an empty tree
	EmptyTree() ObjectID
	// FullLength is the length of the hash's hex string
	FullLength() int
	// IsValid returns true if the input is a valid hash
	IsValid(input string) bool
	// MustID creates a new ObjectID from a byte slice
	MustID(b []byte) ObjectID
	// ComputeHash compute the hash for a given ObjectType and content
	ComputeHash(t ObjectType, content []byte) ObjectID
}

// Sha1ObjectFormatImpl implements ObjectFormat interface
type Sha1ObjectFormatImpl struct{}

var (
	emptySha1ObjectID = &Sha1Hash{}
	emptySha1Tree     = &Sha1Hash{
		0x4b, 0x82, 0x5d, 0xc6, 0x42, 0xcb, 0x6e, 0xb9, 0xa0, 0x60,
		0xe5, 0x4b, 0xf8, 0xd6, 0x92, 0x88, 0xfb, 0xee, 0x49, 0x04,
	}
)

// Name получаем имя sha1
func (Sha1ObjectFormatImpl) Name() string { return "sha1" }

// EmptyObjectID возвращаем пустой sha1 объект
func (Sha1ObjectFormatImpl) EmptyObjectID() ObjectID {
	return emptySha1ObjectID
}

// EmptyTree возвращаем пустое дерево sha1 объекта
func (Sha1ObjectFormatImpl) EmptyTree() ObjectID {
	return emptySha1Tree
}

// FullLength возвращаем длину sha1 хеша в виде 40 символов (sha1 = 40 символов)
func (Sha1ObjectFormatImpl) FullLength() int { return 40 }

// IsValid проверяет, является ли входная строка валидным sha1 хешем
func (Sha1ObjectFormatImpl) IsValid(input string) bool {
	return sha1Pattern.MatchString(input)
}

// MustID возвращает объект id
func (Sha1ObjectFormatImpl) MustID(b []byte) ObjectID {
	var id Sha1Hash
	copy(id[0:20], b)
	return &id
}

// ComputeHash compute the hash for a given ObjectType and content
func (h Sha1ObjectFormatImpl) ComputeHash(t ObjectType, content []byte) ObjectID {
	hasher := sha1.New()
	_, err := hasher.Write(t.Bytes())
	if err != nil {
		log.Error("Error has occurred while writing hash by object type for sha1: %v", err)
		return nil
	}
	_, err = hasher.Write([]byte(" "))
	if err != nil {
		log.Error("Error has occurred while writing hash with space for sha1: %v", err)
		return nil
	}
	_, err = hasher.Write([]byte(strconv.FormatInt(int64(len(content)), 10)))
	if err != nil {
		log.Error("Error has occurred while writing hash by length of content for sha1: %v", err)
		return nil
	}
	_, err = hasher.Write([]byte{0})
	if err != nil {
		log.Error("Error has occurred while writing hash with zero value for sha1: %v", err)
		return nil
	}
	_, err = hasher.Write(content)
	if err != nil {
		log.Error("Error has occurred while writing hash with content for sha1: %v", err)
		return nil
	}
	return h.MustID(hasher.Sum(nil))
}

type Sha256ObjectFormatImpl struct{}

var (
	emptySha256ObjectID = &Sha256Hash{}
	emptySha256Tree     = &Sha256Hash{
		0x6e, 0xf1, 0x9b, 0x41, 0x22, 0x5c, 0x53, 0x69, 0xf1, 0xc1,
		0x04, 0xd4, 0x5d, 0x8d, 0x85, 0xef, 0xa9, 0xb0, 0x57, 0xb5,
		0x3b, 0x14, 0xb4, 0xb9, 0xb9, 0x39, 0xdd, 0x74, 0xde, 0xcc,
		0x53, 0x21,
	}
)

// Name получаем имя sha256
func (Sha256ObjectFormatImpl) Name() string { return "sha256" }

// EmptyObjectID возвращаем пустой sha256 объект
func (Sha256ObjectFormatImpl) EmptyObjectID() ObjectID {
	return emptySha256ObjectID
}

// EmptyTree возвращаем пустое дерево sha256 объекта
func (Sha256ObjectFormatImpl) EmptyTree() ObjectID {
	return emptySha256Tree
}

// FullLength возвращаем длину sha1 хеша в виде 64 символов (sha256 = 64 символов)
func (Sha256ObjectFormatImpl) FullLength() int { return 64 }

// IsValid проверяет, является ли входная строка валидным sha256 хешем
func (Sha256ObjectFormatImpl) IsValid(input string) bool {
	return sha256Pattern.MatchString(input)
}

// MustID возвращает объект id
func (Sha256ObjectFormatImpl) MustID(b []byte) ObjectID {
	var id Sha256Hash
	copy(id[0:32], b)
	return &id
}

// ComputeHash compute the hash for a given ObjectType and content
func (h Sha256ObjectFormatImpl) ComputeHash(t ObjectType, content []byte) ObjectID {
	hasher := sha256.New()
	_, err := hasher.Write(t.Bytes())
	if err != nil {
		log.Error("Error has occurred while writing hash by object type for sha256: %v", err)
		return nil
	}
	_, err = hasher.Write([]byte(" "))
	if err != nil {
		log.Error("Error has occurred while writing hash with space for sha256: %v", err)
		return nil
	}
	_, err = hasher.Write([]byte(strconv.FormatInt(int64(len(content)), 10)))
	if err != nil {
		log.Error("Error has occurred while writing hash by length of content for sha256: %v", err)
		return nil
	}
	_, err = hasher.Write([]byte{0})
	if err != nil {
		log.Error("Error has occurred while writing hash with zero value for sha256: %v", err)
		return nil
	}
	_, err = hasher.Write(content)
	if err != nil {
		log.Error("Error has occurred while writing hash with content for sha256: %v", err)
		return nil
	}
	return h.MustID(hasher.Sum(nil))
}

var (
	Sha1ObjectFormat   ObjectFormat = Sha1ObjectFormatImpl{}
	Sha256ObjectFormat ObjectFormat = Sha256ObjectFormatImpl{}
)

// ObjectFormatFromName проверяем поддерживаемый формат объекта
func ObjectFormatFromName(name string) ObjectFormat {
	for _, objectFormat := range DefaultFeatures().SupportedObjectFormats {
		if name == objectFormat.Name() {
			return objectFormat
		}
	}
	return nil
}

// IsValidObjectFormat проверяем валидность переданного объекта
func IsValidObjectFormat(name string) bool {
	return ObjectFormatFromName(name) != nil
}
