package main

import (
	"testing"
)

func TestChecksum(t *testing.T) {
	sample := []byte{
		0x45, 0x00, 0x00, 0x73,
		0x00, 0x00, 0x40, 0x00,
		0x40, 0x11, 0xb8, 0x61,
		0xc0, 0xa8, 0x00, 0x01,
		0xc0, 0xa8, 0x00, 0xc7,
		// useless after this
		0x00, 0x35, 0xe9, 0x7c,
		0x00, 0x5f, 0x27, 0x9f,
		0x1e, 0x4b, 0x81, 0x80,
	}

	checksum := calculateIPHeaderChecksum(sample)
	if checksum != 0xb861 {
		t.Fatalf("Checksum 0x%x didn't match reference 0xb861", checksum)
	}
}