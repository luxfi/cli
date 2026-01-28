// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build tools
// +build tools

package main

import (
	"fmt"

	"golang.org/x/crypto/pbkdf2"
	//"golang.org/x/crypto/hkdf" // check if this exists
)

func main() {
	fmt.Println("Inspecting crypto/sha3")
	// Try to find NewLegacyKeccak256
	// Since we can't use reflection on package, we can only try to compile or print knowns.
	// But wait, if I can import it, I can print it?
	// Go doesn't allow printing package exports easily at runtime without static ref.

	// Let's just print type of pbkdf2.Key
	fmt.Printf("pbkdf2.Key type: %T\n", pbkdf2.Key)

	// Check sha3
	// fmt.Printf("sha3.NewLegacyKeccak256: %T\n", sha3.NewLegacyKeccak256) // Compiler will fail if missing

	// We can rely on compiler error from this file to tell us what is missing.
}
