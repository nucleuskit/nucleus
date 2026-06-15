package gen

// ContractSourceHash is generated from contract source files.
const ContractSourceHash = "6399c0636543078a250f6ee503877d3a75637fe0242950245796a2299874b3a1"

// EmbeddedContractSource is a generated contract source snapshot.
type EmbeddedContractSource struct {
	Path    string
	Content string
}

// EmbeddedContractSources contains generated contract source snapshots.
var EmbeddedContractSources = []EmbeddedContractSource{
	{Path: "api/errors.yaml", Content: "errors:\n  - code: 4001\n    message: invalid_name\n    http_status: 400\n  - code: 4041\n    message: greeting_not_found\n    http_status: 404\n"},
	{Path: "api/openapi.yaml", Content: "openapi: 3.1.0\ninfo:\n  title: Hello HTTP Example\n  version: 0.1.0\npaths:\n  /hello/{name}:\n    parameters:\n      - name: name\n        in: path\n        required: true\n        schema:\n          type: string\n    get:\n      operationId: get_hello\n      x-nucleus-priority: 10\n      parameters:\n        - name: trace_id\n          in: header\n          required: false\n          schema:\n            type: string\n      responses:\n        \"200\":\n          description: Greeting response.\n        \"404\":\n          description: Greeting target was not found.\n"},
}
