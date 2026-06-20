// Package id предоставляет генератор коротких уникальных идентификаторов
// на основе nanoid-алгоритма.
//
// Аналог Python-версии из magutils/src/magutils/id.py
package id

import (
	"errors"

	nanoid "github.com/matoous/go-nanoid/v2"
)

// Значения по умолчанию.
const (
	DefaultAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	DefaultSize     = 15
	MinSize         = 5
	MinAlphabetLen  = 10
)

// Пакетные переменные для конфигурации (аналог ConfigMeta в Python).
var (
	alphabet = DefaultAlphabet
	size     = DefaultSize
)

// SetAlphabet устанавливает алфавит для генерации ID.
// Должен содержать минимум 10 уникальных символов.
func SetAlphabet(alp string) error {
	if len(alp) < MinAlphabetLen {
		return errors.New("alphabet must contain at least 10 characters")
	}
	if !allUnique(alp) {
		return errors.New("characters in the alphabet must be unique")
	}
	alphabet = alp
	return nil
}

// SetSize устанавливает длину генерируемого ID.
// Не может быть меньше MinSize (5).
func SetSize(n int) error {
	if n < MinSize {
		return errors.New("id size cannot be shorter than 5 characters")
	}
	size = n
	return nil
}

// Alphabet возвращает текущий алфавит.
func Alphabet() string {
	return alphabet
}

// Size возвращает текущую длину ID.
func Size() int {
	return size
}

// Gen генерирует новый ID заданной длины из текущего алфавита.
// Использует nanoid-алгоритм (crypto/rand + rejection sampling).
//
// Аналог: nanoid.generate(Config.alphabet, Config.size)
func Gen() (string, error) {
	return nanoid.Generate(alphabet, size)
}

// GenWith генерирует ID указанной длины из указанного алфавита.
func GenWith(alp string, n int) (string, error) {
	if len(alp) < MinAlphabetLen {
		return "", errors.New("alphabet must contain at least 10 characters")
	}
	if n < MinSize {
		return "", errors.New("id size cannot be shorter than 5 characters")
	}
	return nanoid.Generate(alp, n)
}

// MustGen генерирует ID и паникует при ошибке.
// Удобно для инициализации, где ошибка невозможна.
func MustGen() string {
	id, err := Gen()
	if err != nil {
		panic(err)
	}
	return id
}

// IsValid проверяет, что ID соответствует текущим настройкам алфавита и длины.
//
// Аналог: is_valid() из Python.
func IsValid(id string) error {
	if len(id) != size {
		return errors.New("invalid id length")
	}
	alp := make(map[rune]struct{}, len(alphabet))
	for _, ch := range alphabet {
		alp[ch] = struct{}{}
	}
	for _, ch := range id {
		if _, ok := alp[ch]; !ok {
			return errors.New("invalid id: contains characters outside the alphabet")
		}
	}
	return nil
}

// allUnique проверяет, что все символы в строке уникальны.
func allUnique(s string) bool {
	seen := make(map[rune]struct{}, len(s))
	for _, ch := range s {
		if _, ok := seen[ch]; ok {
			return false
		}
		seen[ch] = struct{}{}
	}
	return true
}
