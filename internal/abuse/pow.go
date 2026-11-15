package abuse

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const powAlgorithm = "sha256-leading-zero-bits"

// Challenge is what we hand to clients. Challenge is an opaque signed blob;
// Nonce + Difficulty are duplicated in cleartext so the client can mine.
type Challenge struct {
	Challenge  string    `json:"challenge"`
	Nonce      string    `json:"nonce"`
	Difficulty int       `json:"difficulty"`
	ExpiresAt  time.Time `json:"expiresAt"`
	Algorithm  string    `json:"algorithm"`
}

type Verifier struct {
	Secret     []byte
	Difficulty int
	TTL        time.Duration
}

func NewVerifier(secret string, difficulty int, ttl time.Duration) *Verifier {
	return &Verifier{
		Secret:     []byte(secret),
		Difficulty: difficulty,
		TTL:        ttl,
	}
}

func (v *Verifier) Issue(ipClass string, now time.Time) (Challenge, error) {
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return Challenge{}, err
	}
	idBytes := make([]byte, 12)
	if _, err := rand.Read(idBytes); err != nil {
		return Challenge{}, err
	}

	id := hex.EncodeToString(idBytes)
	nonce := hex.EncodeToString(nonceBytes)
	exp := now.Add(v.TTL)
	token, err := v.encode(id, nonce, v.Difficulty, exp, ipClass)
	if err != nil {
		return Challenge{}, err
	}

	return Challenge{
		Challenge:  token,
		Nonce:      nonce,
		Difficulty: v.Difficulty,
		ExpiresAt:  exp,
		Algorithm:  powAlgorithm,
	}, nil
}

type VerifiedProof struct {
	ID         string
	Nonce      string
	Difficulty int
	ExpiresAt  time.Time
	IPClass    string
}

func (v *Verifier) Verify(challengeToken string, counter uint64, ipClass string, now time.Time) (VerifiedProof, error) {
	p, err := v.decode(challengeToken)
	if err != nil {
		return VerifiedProof{}, err
	}
	if now.After(p.ExpiresAt) {
		return VerifiedProof{}, fmt.Errorf("challenge expired")
	}
	// Prefer IP-bound token; accept unbound tokens issued when IP was unknown.
	if p.IPClass != "" && p.IPClass != ipClass {
		return VerifiedProof{}, fmt.Errorf("challenge ip mismatch")
	}
	if !hasLeadingZeroBits(hashPow(p.Nonce, counter), p.Difficulty) {
		return VerifiedProof{}, fmt.Errorf("proof of work failed")
	}
	return p, nil
}

func (v *Verifier) encode(id, nonce string, difficulty int, exp time.Time, ipClass string) (string, error) {
	payload := strings.Join([]string{
		id,
		nonce,
		strconv.Itoa(difficulty),
		strconv.FormatInt(exp.Unix(), 10),
		ipClass,
	}, "|")
	mac := hmac.New(sha256.New, v.Secret)
	_, _ = mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	raw := payload + "|" + sig
	return base64.RawURLEncoding.EncodeToString([]byte(raw)), nil
}

func (v *Verifier) decode(token string) (VerifiedProof, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return VerifiedProof{}, fmt.Errorf("invalid challenge encoding")
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 6 {
		return VerifiedProof{}, fmt.Errorf("invalid challenge format")
	}
	id, nonce, diffStr, expStr, ipClass, sig := parts[0], parts[1], parts[2], parts[3], parts[4], parts[5]
	difficulty, err := strconv.Atoi(diffStr)
	if err != nil {
		return VerifiedProof{}, fmt.Errorf("invalid difficulty")
	}
	expUnix, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return VerifiedProof{}, fmt.Errorf("invalid expiry")
	}
	payload := strings.Join([]string{id, nonce, diffStr, expStr, ipClass}, "|")
	mac := hmac.New(sha256.New, v.Secret)
	_, _ = mac.Write([]byte(payload))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(sig)) {
		return VerifiedProof{}, fmt.Errorf("invalid challenge signature")
	}
	return VerifiedProof{
		ID:         id,
		Nonce:      nonce,
		Difficulty: difficulty,
		ExpiresAt:  time.Unix(expUnix, 0).UTC(),
		IPClass:    ipClass,
	}, nil
}

func hashPow(nonce string, counter uint64) []byte {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", nonce, counter)))
	return sum[:]
}

func hasLeadingZeroBits(digest []byte, bits int) bool {
	full := bits / 8
	rem := bits % 8
	for i := 0; i < full; i++ {
		if digest[i] != 0 {
			return false
		}
	}
	if rem == 0 {
		return true
	}
	mask := byte(0xFF << (8 - rem))
	return digest[full]&mask == 0
}

// Solve is used in tests.
func Solve(nonce string, difficulty int) (uint64, error) {
	for c := uint64(0); c < 1<<32; c++ {
		if hasLeadingZeroBits(hashPow(nonce, c), difficulty) {
			return c, nil
		}
	}
	return 0, fmt.Errorf("no solution found")
}

// IPClass reduces an IP to a coarse bucket so attestations survive CGNAT churn.
func IPClass(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	if strings.Contains(ip, ":") {
		parts := strings.Split(ip, ":")
		if len(parts) >= 4 {
			return strings.Join(parts[:4], ":")
		}
		return ip
	}
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return parts[0] + "." + parts[1] + "." + parts[2]
	}
	return ip
}
