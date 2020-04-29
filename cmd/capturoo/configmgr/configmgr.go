package configmgr

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"capturoo-cli-tool-go/fbauth"

	"github.com/pkg/errors"
)

const (
	configDir = ".capturoo"
)

var (
	// ErrTokenFileNotFound error
	ErrTokenFileNotFound = errors.New("token file not found")
)

// JWTData holds Firebase JWT structured data.
type JWTData struct {
	Name      string `json:"name,omitempty"`
	CapAID    string `json:"cap_aid,omitempty"`
	CapRole   string `json:"cap_role,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	Email     string `json:"email,omitempty"`
	Audience  string `json:"aud,omitempty"`
	ExpiresAt int64  `json:"exp,omitempty"`
	IssuedAt  int64  `json:"iat,omitempty"`
	Issuer    string `json:"iss,omitempty"`
	Subject   string `json:"sub,omitempty"`
}

// User struct
type User struct {
	UID         string `mapstructure:"uid" yaml:"uid"`
	Email       string `mapstructure:"email" yaml:"email"`
	Role        string `mapstructure:"role" yaml:"role"`
	DisplayName string `mapstructure:"displayName" yaml:"displayName"`
}

// ErrTokenExpired sentinel value
var ErrTokenExpired error = errors.New("token-expired")

func myTest() {
	fbRESTClient := fbauth.NewRESTClient()

	fmt.Printf("%#v\n", fbRESTClient)
}

// ReadTokenAndRefreshToken reads the token and refresh token from the filesystem
// or returns nil if the file has not yet been created.
func ReadTokenAndRefreshToken(filename string) (*fbauth.TokenAndRefreshToken, error) {
	hd, err := homeDir()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get home directory")
	}
	filepath := filepath.Join(hd, configDir, filename)

	exists, err := exists(filepath)
	if err != nil {
		return nil, errors.Wrapf(err, "exists(path=%q) failed", filepath)
	}
	if !exists {
		return nil, fmt.Errorf("token file %q not found: %w", filename, ErrTokenFileNotFound)
	}

	f, err := os.Open(filepath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open file %q", filepath)
	}
	defer f.Close()

	tart := &fbauth.TokenAndRefreshToken{}
	if err := json.NewDecoder(f).Decode(tart); err != nil {
		return nil, err
	}

	expired, err := isTokenExpired(tart)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if token expired")
	}
	if expired {
		return tart, ErrTokenExpired
	}
	return tart, nil
}

func isTokenExpired(tar *fbauth.TokenAndRefreshToken) (bool, error) {
	jwtData, err := ParseJWT(tar.IDToken)
	if err != nil {
		return false, errors.Wrap(err, "parse jwt")
	}

	utcNow := time.Now().Unix()
	if jwtData.ExpiresAt-utcNow <= 0 {
		return true, nil
	}
	return false, nil
}

//ParseJWT extracts the data section without any verification.
func ParseJWT(jwt string) (*JWTData, error) {
	parts := strings.Split(jwt, ".")
	p2, err := func(seg string) ([]byte, error) {
		if l := len(seg) % 4; l > 0 {
			seg += strings.Repeat("=", 4-l)
		}
		return base64.URLEncoding.DecodeString(seg)
	}(parts[1])
	if err != nil {
		return nil, err

	}

	var data JWTData
	if err := json.Unmarshal(p2, &data); err != nil {
		return nil, errors.Wrapf(err, "failed to decode JWT data")
	}
	return &data, nil
}

// WriteTokenAndRefreshToken writes a copy of the token and refresh token to file.
func WriteTokenAndRefreshToken(filename string, tar *fbauth.TokenAndRefreshToken) error {
	if tar == nil {
		return fmt.Errorf("token and refresh token was nil")
	}

	err := ensureConfigDirExists()
	if err != nil {
		return errors.Wrapf(err, "couldn't ensure config dir exists")
	}
	hd, err := homeDir()
	if err != nil {
		return errors.Wrap(err, "failed to get home directory")
	}
	filepath := filepath.Join(hd, configDir, filename)

	f, err := os.Create(filepath)
	if err != nil {
		return errors.Wrapf(err, "create file %q", filepath)
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(tar)
	if err != nil {
		return errors.Wrap(err, "json encode token")
	}
	return nil
}

func ensureConfigDirExists() error {
	hd, err := homeDir()
	if err != nil {
		return errors.Wrap(err, "failed homeDir()")
	}
	cfgDir := filepath.Join(hd, configDir)

	exists, err := exists(cfgDir)
	if err != nil {
		return errors.Wrapf(err, "failed exists(%s)", configDir)
	}
	if exists {
		return nil
	}

	os.Mkdir(cfgDir, 0755)
	return nil
}

func homeDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", errors.Wrap(err, "user.Current()")
	}
	return usr.HomeDir, nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
