package tarkovdev

import (
	"net/http"
)

type TarkovDevClient struct {
	baseURL string
	client  *http.Client
}

func New() *TarkovDevClient {
	return &TarkovDevClient{
		baseURL: "https://api.tarkov-dev.com/graphql",
		client:  http.DefaultClient,
	}
}
