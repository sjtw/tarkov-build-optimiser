package tarkovdev

import (
	"net/http"

	"github.com/Khan/genqlient/graphql"
)

type Api struct {
	baseURL string
	http    *http.Client
	Client  graphql.Client
}

func New() *Api {
	client := graphql.NewClient("https://api.tarkov.dev/graphql", http.DefaultClient)

	return &Api{
		baseURL: "https://api.tarkov-dev.com/graphql",
		http:    http.DefaultClient,
		Client:  client,
	}
}
