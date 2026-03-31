package password

import "golang.org/x/crypto/bcrypt"

func Hash(raw string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func Compare(hashed, raw string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(raw))
}
