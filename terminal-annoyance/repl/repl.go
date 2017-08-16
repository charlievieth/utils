package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

var inFile = bytes.NewReader([]byte(`It was the best of times, it was the worst of times, it was the age of wisdom, it was the age of foolishness, it was the epoch of belief, it was the epoch of incredulity, it was the season of Light, it was the season of Darkness, it was the spring of hope, it was the winter of despair, we had everything before us, we had nothing before us, we were all going direct to Heaven, we were all going direct the other way - in short, the period was so far like the present period, that some of its noisiest authorities insisted on its being received, for good or for evil, in the superlative degree of comparison only.`))

func main() {
	var x [aes.BlockSize]byte
	if _, err := rand.Read(x[0:]); err != nil {
		panic(err)
	}

	base := sha256.Sum256([]byte("example key 1234"))
	key := base[:]

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// If the key is unique for each ciphertext, then it's ok to use a zero
	// IV.
	var iv [aes.BlockSize]byte
	_ = iv
	// stream := cipher.NewOFB(block, iv[:])
	stream := cipher.NewOFB(block, x[0:])

	outFile := new(bytes.Buffer)

	writer := &cipher.StreamWriter{S: stream, W: outFile}
	// Copy the input file to the output file, encrypting as we go.
	if _, err := io.Copy(writer, inFile); err != nil {
		panic(err)
	}

	// stream = cipher.NewOFB(block, iv[:])
	stream = cipher.NewOFB(block, x[0:])
	reader := &cipher.StreamReader{S: stream, R: outFile}
	src := new(bytes.Buffer)

	if _, err := io.Copy(src, reader); err != nil {
		panic(err)
	}

	fmt.Println(src.String())

	// Note that this example is simplistic in that it omits any
	// authentication of the encrypted data. If you were actually to use
	// StreamReader in this manner, an attacker could flip arbitrary bits in
	// the decrypted result.
}
