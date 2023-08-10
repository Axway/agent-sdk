package oauth

import "testing"

func TestGetKey(t *testing.T) {
	cases := []struct {
		description string
		kr          *keyReader
	}{
		{
			"no password",
			&keyReader{
				privKey: "../../apic/auth/testdata/private_key.pem",
			},
		},
		{
			"with empty password file",
			&keyReader{
				privKey:  "../../apic/auth/testdata/private_key.pem",
				password: "../../apic/auth/testdata/password_empty",
			},
		},
		{
			"with password",
			&keyReader{
				privKey:  "../../apic/auth/testdata/private_key_with_pwd.pem",
				password: "../../apic/auth/testdata/password",
			},
		},
	}

	for _, testCase := range cases {
		if _, err := testCase.kr.GetPrivateKey(); err != nil {
			t.Errorf("testcase: %s: failed to read rsa key %s", testCase.description, err)
		}
	}
}

func TestGetPublicKey(t *testing.T) {
	cases := []struct {
		description string
		kr          *keyReader
	}{
		{
			"with public key",
			&keyReader{
				publicKey: "../../apic/auth/testdata/public_key",
			},
		},
		{
			"with private and public key",
			&keyReader{
				privKey:   "../../apic/auth/testdata/private_key.pem",
				publicKey: "../../apic/auth/testdata/public_key",
			},
		},
		{
			"with private, public key, and password",
			&keyReader{
				privKey:   "../../apic/auth/testdata/private_key_with_pwd.pem",
				password:  "../../apic/auth/testdata/password",
				publicKey: "../../apic/auth/testdata/public_key",
			},
		},
	}
	for _, testCase := range cases {
		if _, err := testCase.kr.GetPublicKey(); err != nil {
			t.Errorf("testcase: %s: failed to read public key %s", testCase.description, err)
		}
	}
}
