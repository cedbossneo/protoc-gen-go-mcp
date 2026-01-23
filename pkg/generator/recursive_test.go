package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestMessageSchema_RecursiveType(t *testing.T) {
	// Create a recursive message descriptor:
	// message RecursiveNode {
	//   string value = 1;
	//   RecursiveNode next = 2;
	// }
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("recursive.proto"),
		Package: proto.String("test"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("RecursiveNode"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("value"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						JsonName: proto.String("value"),
					},
					{
						Name:     proto.String("next"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".test.RecursiveNode"),
						JsonName: proto.String("next"),
					},
				},
			},
		},
	}

	files, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{fdp},
	})
	if err != nil {
		t.Fatalf("failed to create files: %v", err)
	}

	desc, err := files.FindDescriptorByName("test.RecursiveNode")
	if err != nil {
		t.Fatalf("failed to find message descriptor: %v", err)
	}
	md := desc.(protoreflect.MessageDescriptor)

	g := &FileGenerator{}

	// Test standard schema generation (should not panic/stackoverflow)
	schema := g.messageSchema(md, make(map[string]struct{}))

	// Basic assertions to verify the schema structure
	assert.Equal(t, "object", schema["type"])
	properties := schema["properties"].(map[string]any)
	assert.Contains(t, properties, "value")
	assert.Contains(t, properties, "next")

	// Check that the recursive field is handled gracefully (empty object or simplified schema)
	nextField := properties["next"].(map[string]any)
	// Based on our implementation, when it hits recursion, it returns map[string]any{"type": "object"}
	// But since it's the first level of recursion, it might still have properties if we haven't visited it yet.
	// Wait, top level visits "test.RecursiveNode".
	// Then field "next" is "test.RecursiveNode".
	// So inside properties["next"], we call getType -> messageSchema("test.RecursiveNode").
	// Since "test.RecursiveNode" is already in visited, it should return {"type": "object"}.

	assert.Equal(t, "object", nextField["type"])
	// It shouldn't have properties because of the early exit
	assert.NotContains(t, nextField, "properties")

	// Test OpenAI schema generation
	g.openAICompat = true
	schemaOpenAI := g.messageSchema(md, make(map[string]struct{}))

	// Similar assertions for OpenAI mode
	// In OpenAI mode, type might be ["object", "null"] or "object" depending on context,
	// but the recursive break returns {"type": "object"} directly.

	propertiesOpenAI := schemaOpenAI["properties"].(map[string]any)
	nextFieldOpenAI := propertiesOpenAI["next"].(map[string]any)

	// The recursive break returns {"type": "object"}
	// However, the calling code in getType might wrap or modify it?
	// In getType:
	// schema = g.messageSchema(fd.Message(), visited)
	// It returns exactly what messageSchema returns.

	assert.Equal(t, "object", nextFieldOpenAI["type"])
	assert.NotContains(t, nextFieldOpenAI, "properties")
}
