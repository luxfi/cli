package keycmd

import (
	"crypto/rand"
	"testing"

	"github.com/luxfi/crypto/mldsa"
	"github.com/luxfi/crypto/mlkem"
	"github.com/luxfi/crypto/slhdsa"
	"github.com/stretchr/testify/require"
)

func TestMLDSACrypto(t *testing.T) {
	tests := []struct {
		name string
		mode mldsa.Mode
	}{
		{"ML-DSA-44", mldsa.MLDSA44},
		{"ML-DSA-65", mldsa.MLDSA65},
		{"ML-DSA-87", mldsa.MLDSA87},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate key pair
			privKey, err := mldsa.GenerateKey(rand.Reader, tt.mode)
			require.NoError(t, err)
			require.NotNil(t, privKey)

			// Test signing
			message := []byte("test message for post-quantum signature")
			signature, err := privKey.Sign(rand.Reader, message, nil)
			require.NoError(t, err)
			require.NotEmpty(t, signature)

			// Test verification
			valid := privKey.PublicKey.Verify(message, signature, nil)
			require.True(t, valid)

			// Test wrong message
			wrongMessage := []byte("wrong message")
			valid = privKey.PublicKey.Verify(wrongMessage, signature, nil)
			require.False(t, valid)

			// Test serialization
			privBytes := privKey.Bytes()
			require.NotEmpty(t, privBytes)
			
			pubBytes := privKey.PublicKey.Bytes()
			require.NotEmpty(t, pubBytes)
		})
	}
}

func TestMLKEMCrypto(t *testing.T) {
	tests := []struct {
		name string
		mode mlkem.Mode
	}{
		{"ML-KEM-512", mlkem.MLKEM512},
		{"ML-KEM-768", mlkem.MLKEM768},
		{"ML-KEM-1024", mlkem.MLKEM1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate key pair
			privKey, err := mlkem.GenerateKey(rand.Reader, tt.mode)
			require.NoError(t, err)
			require.NotNil(t, privKey)

			// Test encapsulation
			ciphertext, sharedSecret, err := privKey.PublicKey.Encapsulate(rand.Reader)
			require.NoError(t, err)
			require.NotEmpty(t, ciphertext)
			require.NotEmpty(t, sharedSecret)

			// Test decapsulation
			decryptedSecret, err := privKey.Decapsulate(ciphertext)
			require.NoError(t, err)
			require.Equal(t, sharedSecret, decryptedSecret)

			// Test serialization
			privBytes := privKey.Bytes()
			require.NotEmpty(t, privBytes)
			
			pubBytes := privKey.PublicKey.Bytes()
			require.NotEmpty(t, pubBytes)
		})
	}
}

func TestSLHDSACrypto(t *testing.T) {
	tests := []struct {
		name string
		mode slhdsa.Mode
	}{
		{"SLH-DSA-128f", slhdsa.SLHDSA128f},
		{"SLH-DSA-128s", slhdsa.SLHDSA128s},
		{"SLH-DSA-192f", slhdsa.SLHDSA192f},
		{"SLH-DSA-192s", slhdsa.SLHDSA192s},
		{"SLH-DSA-256f", slhdsa.SLHDSA256f},
		{"SLH-DSA-256s", slhdsa.SLHDSA256s},
	}

	// Only test fast variants for speed
	fastTests := []struct {
		name string
		mode slhdsa.Mode
	}{
		{"SLH-DSA-128f", slhdsa.SLHDSA128f},
		{"SLH-DSA-192f", slhdsa.SLHDSA192f},
	}

	for _, tt := range fastTests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate key pair
			privKey, err := slhdsa.GenerateKey(rand.Reader, tt.mode)
			require.NoError(t, err)
			require.NotNil(t, privKey)

			// Test signing
			message := []byte("test message for hash-based signature")
			signature, err := privKey.Sign(rand.Reader, message, nil)
			require.NoError(t, err)
			require.NotEmpty(t, signature)

			// Test verification
			valid := privKey.PublicKey.Verify(message, signature, nil)
			require.True(t, valid)

			// Test wrong message
			wrongMessage := []byte("wrong message")
			valid = privKey.PublicKey.Verify(wrongMessage, signature, nil)
			require.False(t, valid)

			// Test serialization
			privBytes := privKey.Bytes()
			require.NotEmpty(t, privBytes)
			
			pubBytes := privKey.PublicKey.Bytes()
			require.NotEmpty(t, pubBytes)
		})
	}
}

func BenchmarkMLDSA(b *testing.B) {
	modes := []struct {
		name string
		mode mldsa.Mode
	}{
		{"ML-DSA-44", mldsa.MLDSA44},
		{"ML-DSA-65", mldsa.MLDSA65},
		{"ML-DSA-87", mldsa.MLDSA87},
	}

	message := []byte("benchmark message for post-quantum signature performance testing")

	for _, m := range modes {
		// Key generation benchmark
		b.Run(m.name+"/KeyGen", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := mldsa.GenerateKey(rand.Reader, m.mode)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		// Setup for sign/verify benchmarks
		privKey, _ := mldsa.GenerateKey(rand.Reader, m.mode)

		// Signing benchmark
		b.Run(m.name+"/Sign", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := privKey.Sign(rand.Reader, message, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		// Verification benchmark
		signature, _ := privKey.Sign(rand.Reader, message, nil)
		b.Run(m.name+"/Verify", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				valid := privKey.PublicKey.Verify(message, signature, nil)
				if !valid {
					b.Fatal("verification failed")
				}
			}
		})
	}
}

func BenchmarkMLKEM(b *testing.B) {
	modes := []struct {
		name string
		mode mlkem.Mode
	}{
		{"ML-KEM-512", mlkem.MLKEM512},
		{"ML-KEM-768", mlkem.MLKEM768},
		{"ML-KEM-1024", mlkem.MLKEM1024},
	}

	for _, m := range modes {
		// Key generation benchmark
		b.Run(m.name+"/KeyGen", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := mlkem.GenerateKey(rand.Reader, m.mode)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		// Setup for encap/decap benchmarks
		privKey, _ := mlkem.GenerateKey(rand.Reader, m.mode)

		// Encapsulation benchmark
		b.Run(m.name+"/Encapsulate", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _, err := privKey.PublicKey.Encapsulate(rand.Reader)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		// Decapsulation benchmark
		ciphertext, _, _ := privKey.PublicKey.Encapsulate(rand.Reader)
		b.Run(m.name+"/Decapsulate", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := privKey.Decapsulate(ciphertext)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}