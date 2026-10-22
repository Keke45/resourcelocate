package auth

import "testing"

func TestPasswordRoundTrip(t *testing.T) {
	h, err := HashPassword("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(h, "correct horse") {
		t.Fatal("right password rejected")
	}
	if CheckPassword(h, "wrong horse") {
		t.Fatal("wrong password accepted")
	}
	if CheckPassword("garbage", "correct horse") {
		t.Fatal("malformed hash accepted")
	}
}
