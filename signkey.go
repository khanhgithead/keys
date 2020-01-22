package keys

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/nacl/sign"
)

// SignPrivateKeySize is the size of the SignKey private key bytes.
const SignPrivateKeySize = 64

// SignPublicKeySize is the size of the SignKey public key bytes.
const SignPublicKeySize = 32

// SignKeySeedSize is the size of the SignKey seed bytes.
const SignKeySeedSize = 32

// SignPublicKey is the public part of sign key pair.
type SignPublicKey struct {
	id        ID
	publicKey *[SignPublicKeySize]byte
}

// SignKeyType (Ed25519).
const SignKeyType KeyType = "kpe"

// SignKey a public/private boxKey which can sign and verify.
type SignKey struct {
	privateKey *[SignPrivateKeySize]byte
	publicKey  *SignPublicKey
}

// NewSignKeyFromPrivateKey constructs SignKey from a private key.
// The public key is derived from the private key.
func NewSignKeyFromPrivateKey(privateKey []byte) (*SignKey, error) {
	if len(privateKey) != SignPrivateKeySize {
		return nil, errors.Errorf("invalid private key length %d", len(privateKey))
	}

	// Derive public key from private key
	edpk := ed25519.PrivateKey(privateKey)
	publicKey := edpk.Public().(ed25519.PublicKey)
	if len(publicKey) != SignPublicKeySize {
		return nil, errors.Errorf("invalid public key bytes (len=%d)", len(publicKey))
	}

	var privateKeyBytes [SignPrivateKeySize]byte
	copy(privateKeyBytes[:], privateKey[:SignPrivateKeySize])

	var publicKeyBytes [SignPublicKeySize]byte
	copy(publicKeyBytes[:], publicKey[:SignPublicKeySize])

	return &SignKey{
		privateKey: &privateKeyBytes,
		publicKey: &SignPublicKey{
			id:        MustID(string(SignKeyType), publicKeyBytes[:]),
			publicKey: &publicKeyBytes,
		},
	}, nil
}

// BoxKey converts SignKey to BoxKey.
func (k *SignKey) BoxKey() *BoxKey {
	secretKey := ed25519PrivateKeyToCurve25519(ed25519.PrivateKey(k.privateKey[:]))
	if len(secretKey) != 32 {
		panic("failed to convert key: invalid secret key bytes")
	}
	return NewBoxKeyFromPrivateKey(Bytes32(secretKey))
}

// NewSignPublicKey creates a SignPublicKey.
func NewSignPublicKey(b *[SignPublicKeySize]byte) *SignPublicKey {
	return &SignPublicKey{
		id:        MustID(string(SignKeyType), b[:]),
		publicKey: b,
	}
}

// SigchainPublicKeyFromID converts ID to SigchainPublicKey.
func SigchainPublicKeyFromID(id ID) (SigchainPublicKey, error) {
	return SignPublicKeyFromID(id)
}

// SignPublicKeyFromID converts ID to SignPublicKey.
func SignPublicKeyFromID(id ID) (*SignPublicKey, error) {
	hrp, b, err := id.Decode()
	if err != nil {
		return nil, err
	}
	if hrp != string(SignKeyType) {
		return nil, errors.Errorf("invalid key type")
	}
	if len(b) != SignPublicKeySize {
		return nil, errors.Errorf("invalid sign public key bytes")
	}
	return &SignPublicKey{
		id:        id,
		publicKey: Bytes32(b),
	}, nil
}

// ID for sign public key.
func (s SignPublicKey) ID() ID {
	return s.id
}

func (s SignPublicKey) String() string {
	return s.id.String()
}

// Bytes for public key.
func (s SignPublicKey) Bytes() *[SignPublicKeySize]byte {
	return s.publicKey
}

// BoxPublicKey converts the ed25519 public key to a curve25519 public key.
func (s SignPublicKey) BoxPublicKey() *BoxPublicKey {
	edpk := ed25519.PublicKey(s.publicKey[:])
	bpk := ed25519PublicKeyToCurve25519(edpk)
	if len(bpk) != 32 {
		panic("unable to convert key: invalid public key bytes")
	}
	return NewBoxPublicKey(Bytes32(bpk))
}

// Verify verifies a message and signature with public key.
func (s SignPublicKey) Verify(b []byte) ([]byte, error) {
	if l := len(b); l < sign.Overhead {
		return nil, errors.Errorf("not enough data for signature")
	}
	_, ok := sign.Open(nil, b, s.publicKey)
	if !ok {
		return nil, errors.Errorf("verify failed")
	}
	return b[sign.Overhead:], nil
}

// VerifyDetached verifies a detached message.
func (s SignPublicKey) VerifyDetached(sig []byte, b []byte) error {
	if len(sig) != sign.Overhead {
		return errors.Errorf("invalid sig bytes length")
	}
	if len(b) == 0 {
		return errors.Errorf("no bytes")
	}
	msg := bytesJoin(sig, b)
	_, err := s.Verify(msg)
	return err
}

// NewSignKeyFromSeed constructs SignKey from an ed25519 seed.
// The private key is derived from this seed and the public key is derived from the private key.
func NewSignKeyFromSeed(seed *[SignKeySeedSize]byte) *SignKey {
	privateKey := ed25519.NewKeyFromSeed(seed[:])
	sk, err := NewSignKeyFromPrivateKey(privateKey)
	if err != nil {
		panic(err)
	}
	return sk
}

// Seed returns information on how to generate this key from ed25519 package seed.
func (k SignKey) Seed() *[SignKeySeedSize]byte {
	pk := ed25519.PrivateKey(k.privateKey[:])
	return Bytes32(pk.Seed())
}

// ID ...
func (k SignKey) ID() ID {
	return k.publicKey.ID()
}

func (k SignKey) String() string {
	return k.publicKey.String()
}

// PublicKey returns public part.
func (k SignKey) PublicKey() *SignPublicKey {
	return k.publicKey
}

// PrivateKey returns private key part.
func (k SignKey) PrivateKey() *[SignPrivateKeySize]byte {
	return k.privateKey
}

// Sign bytes with the (sign) private key.
func (k *SignKey) Sign(b []byte) []byte {
	return Sign(b, k)
}

// SignDetached sign bytes detached.
func (k *SignKey) SignDetached(b []byte) []byte {
	return SignDetached(b, k)
}

// Sign bytes.
func Sign(b []byte, sk *SignKey) []byte {
	return sign.Sign(nil, b, sk.privateKey)
}

// SignDetached sign bytes detached.
func SignDetached(b []byte, sk *SignKey) []byte {
	return Sign(b, sk)[:sign.Overhead]
}

// GenerateSignKey generates a SignKey (Ed25519).
func GenerateSignKey() *SignKey {
	logger.Infof("Generating ed25519 key...")
	seed := Rand32()
	return NewSignKeyFromSeed(seed)
}
