package main

import "testing"

func TestCheckPassword(t *testing.T) {
	password := "my-secret-password"

	hash, err := hashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	err = checkPassword(password, hash)
	if err != nil {
		t.Fatal("password should be valid")
	}
}

func TestCheckPasswordWrongPassword(t *testing.T) {
	password := "my-secret-password"

	hash, err := hashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	err = checkPassword("wrong-password", hash)
	if err == nil {
		t.Fatal("wrong password should not be accepted")
	}
}
