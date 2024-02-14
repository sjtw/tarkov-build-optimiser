// Code generated by github.com/Khan/genqlient, DO NOT EDIT.

package tarkovdev

import (
	"context"

	"github.com/Khan/genqlient/graphql"
)

// GetItemsItemsItem includes the requested fields of the GraphQL type Item.
type GetItemsItemsItem struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// GetId returns GetItemsItemsItem.Id, and is useful for accessing the field via an interface.
func (v *GetItemsItemsItem) GetId() string { return v.Id }

// GetName returns GetItemsItemsItem.Name, and is useful for accessing the field via an interface.
func (v *GetItemsItemsItem) GetName() string { return v.Name }

// GetItemsResponse is returned by GetItems on success.
type GetItemsResponse struct {
	Items []GetItemsItemsItem `json:"items"`
}

// GetItems returns GetItemsResponse.Items, and is useful for accessing the field via an interface.
func (v *GetItemsResponse) GetItems() []GetItemsItemsItem { return v.Items }

// The query or mutation executed by GetItems.
const GetItems_Operation = `
query GetItems {
	items {
		id
		name
	}
}
`

func GetItems(
	ctx context.Context,
	client graphql.Client,
) (*GetItemsResponse, error) {
	req := &graphql.Request{
		OpName: "GetItems",
		Query:  GetItems_Operation,
	}
	var err error

	var data GetItemsResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}
