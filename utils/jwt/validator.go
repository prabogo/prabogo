package jwt

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWKSet represents a JSON Web Key Set
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// jwksCacheEntry holds cached JWKS with expiry
type jwksCacheEntry struct {
	jwkSet    *JWKSet
	expiresAt time.Time
}

var (
	jwksCache    = make(map[string]*jwksCacheEntry)
	jwksCacheMu  sync.RWMutex
	jwksCacheTTL = 5 * time.Minute
)

// JWKSClient handles fetching and caching JWKS
type JWKSClient struct {
	jwksURL string
	client  *http.Client
}

// NewJWKSClient creates a new JWKS client with custom URL
func NewJWKSClient(jwksURL string) *JWKSClient {
	return &JWKSClient{
		jwksURL: jwksURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetJWKSet fetches the JWKS from the URL with 5-minute cache
func (c *JWKSClient) GetJWKSet(ctx context.Context) (*JWKSet, error) {
	jwksCacheMu.RLock()
	if entry, ok := jwksCache[c.jwksURL]; ok && time.Now().Before(entry.expiresAt) {
		jwksCacheMu.RUnlock()
		return entry.jwkSet, nil
	}
	jwksCacheMu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, "GET", c.jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwkSet JWKSet
	if err := json.NewDecoder(resp.Body).Decode(&jwkSet); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	jwksCacheMu.Lock()
	jwksCache[c.jwksURL] = &jwksCacheEntry{jwkSet: &jwkSet, expiresAt: time.Now().Add(jwksCacheTTL)}
	jwksCacheMu.Unlock()

	return &jwkSet, nil
}

// GetPublicKey converts JWK to RSA public key
func (jwk *JWK) GetPublicKey() (*rsa.PublicKey, error) {
	if jwk.Kty != "RSA" {
		return nil, fmt.Errorf("unsupported key type: %s", jwk.Kty)
	}

	// Decode the modulus
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode the exponent
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert to big.Int
	n := new(big.Int).SetBytes(nBytes)
	eBig := new(big.Int).SetBytes(eBytes)
	e := int(eBig.Int64())
	if e == 0 {
		return nil, fmt.Errorf("invalid exponent")
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

// ValidateJWTWithURL validates the JWT token with a specific JWKS URL
// Returns (isValid bool, error)
func ValidateJWTWithURL(tokenString, jwksURL string) (bool, error) {
	// Create JWKS client
	jwksClient := NewJWKSClient(jwksURL)

	// Parse the token to get the header and validate
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the kid from the token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid not found in token header")
		}

		// Fetch JWKS
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		jwkSet, err := jwksClient.GetJWKSet(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
		}

		// Find the matching key
		for _, jwk := range jwkSet.Keys {
			if jwk.Kid == kid {
				return jwk.GetPublicKey()
			}
		}

		return nil, fmt.Errorf("key with kid %s not found in JWKS", kid)
	})

	if err != nil {
		return false, fmt.Errorf("failed to parse/validate token: %w", err)
	}

	if !token.Valid {
		return false, fmt.Errorf("token is not valid")
	}

	// Extract claims to check expiration
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, fmt.Errorf("failed to parse claims")
	}

	// Check token expiration
	if exp, ok := claims["exp"].(float64); ok {
		expTime := time.Unix(int64(exp), 0)
		if time.Now().After(expTime) {
			return false, fmt.Errorf("token has expired at %s", expTime.Format(time.RFC3339))
		}
	} else {
		return false, fmt.Errorf("token does not contain expiration claim")
	}

	// Check not before time if present
	if nbf, ok := claims["nbf"].(float64); ok {
		nbfTime := time.Unix(int64(nbf), 0)
		if time.Now().Before(nbfTime) {
			return false, fmt.Errorf("token is not valid before %s", nbfTime.Format(time.RFC3339))
		}
	}

	return true, nil
}

// GetJWTClaimsWithURL validates JWT with specific URL and returns the claims map
func GetJWTClaimsWithURL(tokenString, jwksURL string) (jwt.MapClaims, error) {
	jwksClient := NewJWKSClient(jwksURL)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid not found in token header")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		jwkSet, jwkErr := jwksClient.GetJWKSet(ctx)
		if jwkErr != nil {
			return nil, jwkErr
		}
		for _, jwk := range jwkSet.Keys {
			if jwk.Kid == kid {
				return jwk.GetPublicKey()
			}
		}
		return nil, fmt.Errorf("key not found")
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to parse claims")
	}
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().After(time.Unix(int64(exp), 0)) {
			return nil, fmt.Errorf("token has expired")
		}
	} else {
		return nil, fmt.Errorf("token does not contain expiration claim")
	}
	return claims, nil
}
