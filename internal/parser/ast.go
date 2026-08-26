package parser

// Manifest is the root structure representing all merged HCL files.
type Manifest struct {
	Endpoints []Endpoint `hcl:"endpoint,block"`
}

// Endpoint represents a single HTTP route definition.
type Endpoint struct {
	MethodAndPath string   `hcl:"name,label"`
	Description   *string  `hcl:"description,attr"`
	Pipeline      Pipeline `hcl:"pipeline,block"`
}

// Pipeline represents the ordered execution steps.
type Pipeline struct {
	Respond RespondStep `hcl:"respond,block"`
}

// RespondStep is the terminal step that writes the HTTP response.
type RespondStep struct {
	Status int     `hcl:"status,attr"`
	Body   *string `hcl:"body,attr"`
}
