# CMS GraphQL Client

### Client generation

Download the client generation tool:

```shell
go get github.com/Khan/genqlient
```

In this folder:

```shell
go run github.com/Khan/genqlient
```

### Usage

```go
package my_package

import (
	"context"
	"fmt"
	"github.com/Khan/genqlient/graphql"
	"net/http"
	cms "terraform-provider-hashicups/cms_graphql_client"
)

func main() {
	ctx := context.Background()
	client := graphql.NewClient("https://api.github.com/graphql", http.DefaultClient)
	resp, err := cms.DataSource(ctx, client, "benjaminjkraft")
	fmt.Println(resp.DataSource.Account.Id, err)
}
```