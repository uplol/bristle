package bristle

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/uplol/bristle/proto/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProtoRegistry struct {
	protoregistry.Files

	Descriptors  map[protoreflect.FullName]protoreflect.Descriptor
	MessageTypes map[protoreflect.FullName]protoreflect.MessageType
}

func NewProtoRegistry() *ProtoRegistry {
	p := &ProtoRegistry{
		Descriptors:  map[protoreflect.FullName]protoreflect.Descriptor{},
		MessageTypes: map[protoreflect.FullName]protoreflect.MessageType{},
	}
	return p
}

func (p *ProtoRegistry) RegisterFile(file protoreflect.FileDescriptor) error {
	messages := file.Messages()
	for i := 0; i < messages.Len(); i++ {
		message := messages.Get(i)
		p.Descriptors[message.FullName()] = message
		p.MessageTypes[message.FullName()] = dynamicpb.NewMessageType(message)
	}

	return nil
}

func (p *ProtoRegistry) RegisterPath(path string) error {
	pathInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !pathInfo.IsDir() {
		err = p.registerFromFile(path)
	} else {
		err = p.registerFromDirectory(path)
	}
	return err
}

func (p *ProtoRegistry) registerFromFile(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	fds := &descriptorpb.FileDescriptorSet{}

	err = proto.Unmarshal(b, fds)
	if err != nil {
		return err
	}

	for _, fd := range fds.File {
		file, err := protodesc.NewFile(fd, p)
		if err != nil {
			return err
		}

		err = p.RegisterFile(file)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *ProtoRegistry) registerFromDirectory(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(info.Name(), ".pb") && !info.IsDir() {

			err := p.registerFromFile(path)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (p *ProtoRegistry) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	if strings.HasPrefix(path, "google/") {
		if path == "google/protobuf/timestamp.proto" {
			err := p.RegisterFile(timestamppb.File_google_protobuf_timestamp_proto)
			if err != nil {
				return nil, err
			}
			return timestamppb.File_google_protobuf_timestamp_proto, nil
		}
	} else if strings.HasSuffix(path, "bristle.proto") {
		return v1.File_bristle_proto, nil
	}

	return p.Files.FindFileByPath(path)
}

func (p *ProtoRegistry) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	result, ok := p.Descriptors[name]
	if !ok {
		return nil, protoregistry.NotFound
	}
	return result, nil
}
