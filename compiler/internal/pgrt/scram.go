package pgrt

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const scramSHA256Mechanism = "SCRAM-SHA-256"

var ErrSCRAMAuthentication = errors.New("SCRAM-SHA-256 authentication failed")

type scramSHA256Client struct {
	user               string
	password           string
	nonce              string
	clientFirstBare    string
	serverFirst        string
	clientFinalBare    string
	serverSignature    []byte
	finalMessageReady  bool
	serverFinalChecked bool
}

func newSCRAMSHA256Client(user string, password string, nonce string) (*scramSHA256Client, error) {
	if nonce == "" || strings.Contains(nonce, ",") {
		return nil, fmt.Errorf("%w: invalid client nonce", ErrSCRAMAuthentication)
	}
	client := &scramSHA256Client{user: user, password: password, nonce: nonce}
	client.clientFirstBare = "n=" + scramEscapeName(user) + ",r=" + nonce
	return client, nil
}

func newSCRAMSHA256ClientWithRandomNonce(user string, password string) (*scramSHA256Client, error) {
	var raw [18]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return nil, err
	}
	return newSCRAMSHA256Client(user, password, base64.RawStdEncoding.EncodeToString(raw[:]))
}

func (c *scramSHA256Client) ClientFirstMessage() string {
	return "n,," + c.clientFirstBare
}

func (c *scramSHA256Client) ClientFinalMessage(serverFirst string) (string, error) {
	fields, err := parseSCRAMAttributes(serverFirst)
	if err != nil {
		return "", err
	}
	serverNonce := fields["r"]
	if serverNonce == "" {
		return "", fmt.Errorf("%w: missing server nonce", ErrMalformedFrame)
	}
	if !strings.HasPrefix(serverNonce, c.nonce) {
		return "", fmt.Errorf(
			"%w: server nonce does not extend client nonce",
			ErrSCRAMAuthentication,
		)
	}
	salt64 := fields["s"]
	if salt64 == "" {
		return "", fmt.Errorf("%w: missing salt", ErrMalformedFrame)
	}
	salt, err := base64.StdEncoding.DecodeString(salt64)
	if err != nil {
		return "", fmt.Errorf("%w: malformed salt", ErrMalformedFrame)
	}
	iterationsRaw := fields["i"]
	if iterationsRaw == "" {
		return "", fmt.Errorf("%w: missing iteration count", ErrMalformedFrame)
	}
	iterations, err := strconv.Atoi(iterationsRaw)
	if err != nil || iterations <= 0 {
		return "", fmt.Errorf("%w: malformed iteration count", ErrMalformedFrame)
	}

	clientFinalWithoutProof := "c=biws,r=" + serverNonce
	authMessage := c.clientFirstBare + "," + serverFirst + "," + clientFinalWithoutProof
	clientProof, serverSignature := scramSHA256ProofAndServerSignature(
		c.password,
		salt,
		iterations,
		authMessage,
	)
	c.serverFirst = serverFirst
	c.clientFinalBare = clientFinalWithoutProof
	c.serverSignature = serverSignature
	c.finalMessageReady = true
	return clientFinalWithoutProof + ",p=" + base64.StdEncoding.EncodeToString(clientProof), nil
}

func (c *scramSHA256Client) VerifyServerFinal(serverFinal string) error {
	if !c.finalMessageReady {
		return fmt.Errorf("%w: client final message was not computed", ErrSCRAMAuthentication)
	}
	fields, err := parseSCRAMAttributes(serverFinal)
	if err != nil {
		return err
	}
	if serverError := fields["e"]; serverError != "" {
		return fmt.Errorf("%w: server final error: %s", ErrSCRAMAuthentication, serverError)
	}
	signature64 := fields["v"]
	if signature64 == "" {
		return fmt.Errorf("%w: missing server signature", ErrMalformedFrame)
	}
	signature, err := base64.StdEncoding.DecodeString(signature64)
	if err != nil {
		return fmt.Errorf("%w: malformed server signature", ErrMalformedFrame)
	}
	if !hmac.Equal(signature, c.serverSignature) {
		return fmt.Errorf("%w: bad server signature", ErrSCRAMAuthentication)
	}
	c.serverFinalChecked = true
	return nil
}

func parseSCRAMAttributes(message string) (map[string]string, error) {
	if message == "" {
		return nil, fmt.Errorf("%w: empty SCRAM message", ErrMalformedFrame)
	}
	fields := map[string]string{}
	for _, part := range strings.Split(message, ",") {
		if len(part) < 3 || part[1] != '=' {
			return nil, fmt.Errorf("%w: malformed SCRAM attribute", ErrMalformedFrame)
		}
		key := part[:1]
		if _, exists := fields[key]; exists {
			return nil, fmt.Errorf("%w: duplicate SCRAM attribute", ErrMalformedFrame)
		}
		fields[key] = part[2:]
	}
	return fields, nil
}

func scramEscapeName(value string) string {
	value = strings.ReplaceAll(value, "=", "=3D")
	value = strings.ReplaceAll(value, ",", "=2C")
	return value
}

func scramSHA256ProofAndServerSignature(
	password string,
	salt []byte,
	iterations int,
	authMessage string,
) ([]byte, []byte) {
	saltedPassword := pbkdf2HMACSHA256([]byte(password), salt, iterations, sha256.Size)
	clientKey := hmacSHA256(saltedPassword, []byte("Client Key"))
	storedKeyHash := sha256.Sum256(clientKey)
	clientSignature := hmacSHA256(storedKeyHash[:], []byte(authMessage))
	clientProof := xorBytes(clientKey, clientSignature)
	serverKey := hmacSHA256(saltedPassword, []byte("Server Key"))
	serverSignature := hmacSHA256(serverKey, []byte(authMessage))
	return clientProof, serverSignature
}

func scramSHA256ServerSignature(
	password string,
	salt []byte,
	iterations int,
	authMessage string,
) []byte {
	_, serverSignature := scramSHA256ProofAndServerSignature(
		password,
		salt,
		iterations,
		authMessage,
	)
	return serverSignature
}

func pbkdf2HMACSHA256(password []byte, salt []byte, iterations int, keyLen int) []byte {
	var out []byte
	for block := 1; len(out) < keyLen; block++ {
		u := hmacSHA256(password, appendInt32(append([]byte(nil), salt...), int32(block)))
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			u = hmacSHA256(password, u)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}

func hmacSHA256(key []byte, message []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(message)
	return mac.Sum(nil)
}

func xorBytes(a []byte, b []byte) []byte {
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}
