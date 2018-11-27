package secrets

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/variables"
)

func encryptAndEncode(encrypter keypairs.Encrypter, value string) (string, *failures.Failure) {
	encrStr, failure := encrypter.EncryptAndEncode([]byte(value))
	if failure != nil {
		return "", variables.FailExpandVariable.New("secrets_err_encrypting", failure.Error())
	}
	return encrStr, nil
}

func decodeAndDecrypt(decrypter keypairs.Decrypter, value string) (string, *failures.Failure) {
	decrBytes, failure := decrypter.DecodeAndDecrypt(value)
	if failure != nil {
		return "", variables.FailExpandVariable.New("secrets_err_decrypting", failure.Error())
	}
	return string(decrBytes), nil
}
