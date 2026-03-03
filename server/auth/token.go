package auth

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sandertv/gophertunnel/minecraft/auth"
	"golang.org/x/oauth2"
)

const tokenFile = "token.json"

func GetTokenSource() oauth2.TokenSource {
	token := new(oauth2.Token)

	data, err := os.ReadFile(tokenFile)
	if err == nil {
		_ = json.Unmarshal(data, token)
		fmt.Println("Используем сохранённый токен")
	} else {
		token, err = auth.RequestLiveToken()
		if err != nil {
			panic("ошибка авторизации: " + err.Error())
		}
	}

	src := auth.RefreshTokenSource(token)
	_, err = src.Token()
	if err != nil {
		fmt.Println("Токен устарел, требуется повторный вход...")
		token, err = auth.RequestLiveToken()
		if err != nil {
			panic("ошибка обновления токена: " + err.Error())
		}
		src = auth.RefreshTokenSource(token)
	}

	// Сохраняем токен
	tok, _ := src.Token()
	b, _ := json.Marshal(tok)
	_ = os.WriteFile(tokenFile, b, 0644)

	return src
}
