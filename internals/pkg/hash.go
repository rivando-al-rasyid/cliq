package pkg

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type HashConfig struct {
	Memory  uint32
	Time    uint32
	Threads uint8
	KeyLen  uint32
	SaltLen uint32
}


func (h *HashConfig) UseRecommended() {
	// based on OWASP min recommendation (May 2023)
	h.Memory = 64 * 1024 // 64 MiB
	h.Time = 2
	h.Threads = 1
	h.KeyLen = 32
	h.SaltLen = 16
}

func (h *HashConfig) genSalt() []byte {
	salt := make([]byte, h.SaltLen)
	// for range h.SaltLen {
	// 	salt = append(salt, rand.Intn())
	// }
	rand.Read(salt)
	return salt
}

func (h *HashConfig) GenHash(pwd string) string {
	salt := h.genSalt()
	// gen hash
	hash := argon2.IDKey([]byte(pwd), salt, h.Time, h.Memory, h.Threads, h.KeyLen)

	version := argon2.Version
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)
	// format hash
	// $argon2id$v=$m=,t=,p=$salt$hash
	out := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", version, h.Memory, h.Time, h.Threads, encodedSalt, encodedHash)
	return out
}

func (h *HashConfig) Compare(pwd string, hashedPwd string) error {
	// n, err := fmt.Sscanf(hashedPwd, "$%s$v=%d$m=%d,t=%d,p=%d$%s$%s", name, version, memory, time, threads, salt, hash)
	// deconstruct hash
	splittedHash := strings.Split(hashedPwd, "$")

	// cek panjang
	if len(splittedHash) != 6 {
		return errors.New("invalid Hash")
	}

	// cek argon2id
	if splittedHash[1] != "argon2id" {
		return errors.New("not argon2id hash")
	}

	// cek versi (v=19)
	var version int
	if _, err := fmt.Sscanf(splittedHash[2], "v=%d", &version); err != nil {
		return errors.New("wrong sscanf syntax")
	}
	if version != argon2.Version {
		return errors.New("wrong argon2id version used")
	}

	// ambil data config m, t, p
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(splittedHash[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return errors.New("wrong sscanf syntax")
	}

	// ambil salt dan hash
	// encodedSalt := splittedHash[4]
	// encodedHash := splittedHash[5]
	salt, err := base64.RawStdEncoding.DecodeString(splittedHash[4])
	if err != nil {
		return errors.New("failed to decode salt")
	}
	hash, err := base64.RawStdEncoding.DecodeString(splittedHash[5])
	if err != nil {
		return errors.New("failed to decode hash")
	}

	// generate hash from incoming password using same config
	newHash := argon2.IDKey([]byte(pwd), salt, time, memory, threads, uint32(len(hash)))

	// compare between hashes
	if subtle.ConstantTimeCompare(hash, newHash) == 0 {
		return errors.New("wrong password")
	}
	return nil
}
