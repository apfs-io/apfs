// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        (unknown)
// source: v1/manifest.proto

package v1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// ManifestTask file processing
type ManifestTask struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id      string    `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Source  string    `protobuf:"bytes,2,opt,name=source,proto3" json:"source,omitempty"`   // '' -> @ = original file
	Target  string    `protobuf:"bytes,3,opt,name=target,proto3" json:"target,omitempty"`   // Name of file
	Type    string    `protobuf:"bytes,4,opt,name=type,proto3" json:"type,omitempty"`       // Target type
	Actions []*Action `protobuf:"bytes,5,rep,name=actions,proto3" json:"actions,omitempty"` // Applied to source before save to target
}

func (x *ManifestTask) Reset() {
	*x = ManifestTask{}
	if protoimpl.UnsafeEnabled {
		mi := &file_v1_manifest_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ManifestTask) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ManifestTask) ProtoMessage() {}

func (x *ManifestTask) ProtoReflect() protoreflect.Message {
	mi := &file_v1_manifest_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ManifestTask.ProtoReflect.Descriptor instead.
func (*ManifestTask) Descriptor() ([]byte, []int) {
	return file_v1_manifest_proto_rawDescGZIP(), []int{0}
}

func (x *ManifestTask) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *ManifestTask) GetSource() string {
	if x != nil {
		return x.Source
	}
	return ""
}

func (x *ManifestTask) GetTarget() string {
	if x != nil {
		return x.Target
	}
	return ""
}

func (x *ManifestTask) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *ManifestTask) GetActions() []*Action {
	if x != nil {
		return x.Actions
	}
	return nil
}

// ManifestTaskStage model object
type ManifestTaskStage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name  string          `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Tasks []*ManifestTask `protobuf:"bytes,2,rep,name=tasks,proto3" json:"tasks,omitempty"`
}

func (x *ManifestTaskStage) Reset() {
	*x = ManifestTaskStage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_v1_manifest_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ManifestTaskStage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ManifestTaskStage) ProtoMessage() {}

func (x *ManifestTaskStage) ProtoReflect() protoreflect.Message {
	mi := &file_v1_manifest_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ManifestTaskStage.ProtoReflect.Descriptor instead.
func (*ManifestTaskStage) Descriptor() ([]byte, []int) {
	return file_v1_manifest_proto_rawDescGZIP(), []int{1}
}

func (x *ManifestTaskStage) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ManifestTaskStage) GetTasks() []*ManifestTask {
	if x != nil {
		return x.Tasks
	}
	return nil
}

// Manifest model object
type Manifest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Version      string               `protobuf:"bytes,1,opt,name=version,proto3" json:"version,omitempty"`
	ContentTypes []string             `protobuf:"bytes,2,rep,name=content_types,json=contentTypes,proto3" json:"content_types,omitempty"`
	Stages       []*ManifestTaskStage `protobuf:"bytes,3,rep,name=stages,proto3" json:"stages,omitempty"`
}

func (x *Manifest) Reset() {
	*x = Manifest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_v1_manifest_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Manifest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Manifest) ProtoMessage() {}

func (x *Manifest) ProtoReflect() protoreflect.Message {
	mi := &file_v1_manifest_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Manifest.ProtoReflect.Descriptor instead.
func (*Manifest) Descriptor() ([]byte, []int) {
	return file_v1_manifest_proto_rawDescGZIP(), []int{2}
}

func (x *Manifest) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *Manifest) GetContentTypes() []string {
	if x != nil {
		return x.ContentTypes
	}
	return nil
}

func (x *Manifest) GetStages() []*ManifestTaskStage {
	if x != nil {
		return x.Stages
	}
	return nil
}

var File_v1_manifest_proto protoreflect.FileDescriptor

var file_v1_manifest_proto_rawDesc = []byte{
	0x0a, 0x11, 0x76, 0x31, 0x2f, 0x6d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x02, 0x76, 0x31, 0x1a, 0x0f, 0x76, 0x31, 0x2f, 0x61, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x88, 0x01, 0x0a, 0x0c, 0x4d, 0x61, 0x6e,
	0x69, 0x66, 0x65, 0x73, 0x74, 0x54, 0x61, 0x73, 0x6b, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x12, 0x16, 0x0a, 0x06, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x06, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x79, 0x70,
	0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x24, 0x0a,
	0x07, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0a,
	0x2e, 0x76, 0x31, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x07, 0x61, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x22, 0x4f, 0x0a, 0x11, 0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x54,
	0x61, 0x73, 0x6b, 0x53, 0x74, 0x61, 0x67, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x26, 0x0a, 0x05,
	0x74, 0x61, 0x73, 0x6b, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x76, 0x31,
	0x2e, 0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x54, 0x61, 0x73, 0x6b, 0x52, 0x05, 0x74,
	0x61, 0x73, 0x6b, 0x73, 0x22, 0x78, 0x0a, 0x08, 0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74,
	0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x23, 0x0a, 0x0d, 0x63, 0x6f,
	0x6e, 0x74, 0x65, 0x6e, 0x74, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28,
	0x09, 0x52, 0x0c, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x73, 0x12,
	0x2d, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x67, 0x65, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x15, 0x2e, 0x76, 0x31, 0x2e, 0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x54, 0x61, 0x73,
	0x6b, 0x53, 0x74, 0x61, 0x67, 0x65, 0x52, 0x06, 0x73, 0x74, 0x61, 0x67, 0x65, 0x73, 0x42, 0x28,
	0x0a, 0x14, 0x63, 0x6f, 0x6d, 0x2e, 0x61, 0x70, 0x66, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x63, 0x6f, 0x6c, 0x2e, 0x76, 0x31, 0x42, 0x08, 0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74,
	0x50, 0x01, 0x5a, 0x04, 0x2e, 0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_v1_manifest_proto_rawDescOnce sync.Once
	file_v1_manifest_proto_rawDescData = file_v1_manifest_proto_rawDesc
)

func file_v1_manifest_proto_rawDescGZIP() []byte {
	file_v1_manifest_proto_rawDescOnce.Do(func() {
		file_v1_manifest_proto_rawDescData = protoimpl.X.CompressGZIP(file_v1_manifest_proto_rawDescData)
	})
	return file_v1_manifest_proto_rawDescData
}

var file_v1_manifest_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_v1_manifest_proto_goTypes = []interface{}{
	(*ManifestTask)(nil),      // 0: v1.ManifestTask
	(*ManifestTaskStage)(nil), // 1: v1.ManifestTaskStage
	(*Manifest)(nil),          // 2: v1.Manifest
	(*Action)(nil),            // 3: v1.Action
}
var file_v1_manifest_proto_depIdxs = []int32{
	3, // 0: v1.ManifestTask.actions:type_name -> v1.Action
	0, // 1: v1.ManifestTaskStage.tasks:type_name -> v1.ManifestTask
	1, // 2: v1.Manifest.stages:type_name -> v1.ManifestTaskStage
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_v1_manifest_proto_init() }
func file_v1_manifest_proto_init() {
	if File_v1_manifest_proto != nil {
		return
	}
	file_v1_action_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_v1_manifest_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ManifestTask); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_v1_manifest_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ManifestTaskStage); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_v1_manifest_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Manifest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_v1_manifest_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_v1_manifest_proto_goTypes,
		DependencyIndexes: file_v1_manifest_proto_depIdxs,
		MessageInfos:      file_v1_manifest_proto_msgTypes,
	}.Build()
	File_v1_manifest_proto = out.File
	file_v1_manifest_proto_rawDesc = nil
	file_v1_manifest_proto_goTypes = nil
	file_v1_manifest_proto_depIdxs = nil
}
