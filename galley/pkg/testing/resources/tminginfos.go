package resources

import "istio.io/istio/galley/pkg/runtime/resource"

var (
	EmptyInfoTming resource.Info

	StructInfoTming resource.Info

	TestSchemaTming *resource.Schema
)

func init123() {
	b := resource.NewSchemaBuilder()

	EmptyInfoTming = b.Register("empty", "type.googleapis.com/google.protobuf.Empty")
	StructInfoTming = b.Register("struct", "type.googleapis.com/google.protobuf.struct")

	TestSchemaTming = b.Build()
}
