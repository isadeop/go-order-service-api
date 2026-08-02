package security

import "testing"

func TestHashPassword_GeneratesVerifiableHash(t *testing.T) {
	hash, err := HashPassword("uma-senha-forte")
	if err != nil {
		t.Fatalf("HashPassword retornou erro inesperado: %v", err)
	}

	if hash == "" {
		t.Fatal("HashPassword retornou hash vazio")
	}

	if hash == "uma-senha-forte" {
		t.Fatal("HashPassword retornou a senha em texto plano, sem hashear")
	}
}

func TestCheckPassword_CorrectPassword(t *testing.T) {
	hash, err := HashPassword("segredo123")
	if err != nil {
		t.Fatalf("HashPassword retornou erro inesperado: %v", err)
	}

	if err := CheckPassword("segredo123", hash); err != nil {
		t.Fatalf("CheckPassword deveria aceitar a senha correta, retornou: %v", err)
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, err := HashPassword("segredo123")
	if err != nil {
		t.Fatalf("HashPassword retornou erro inesperado: %v", err)
	}

	if err := CheckPassword("senha-errada", hash); err == nil {
		t.Fatal("CheckPassword deveria rejeitar uma senha incorreta")
	}
}
