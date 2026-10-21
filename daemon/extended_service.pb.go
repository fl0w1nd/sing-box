package daemon

import (
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ExtensionCapabilities struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Capabilities  []*ExtensionCapability `protobuf:"bytes,1,rep,name=capabilities,proto3" json:"capabilities,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ExtensionCapabilities) Reset() {
	*x = ExtensionCapabilities{}
	mi := &file_daemon_extended_service_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ExtensionCapabilities) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExtensionCapabilities) ProtoMessage() {}

func (x *ExtensionCapabilities) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExtensionCapabilities.ProtoReflect.Descriptor instead.
func (*ExtensionCapabilities) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{0}
}

func (x *ExtensionCapabilities) GetCapabilities() []*ExtensionCapability {
	if x != nil {
		return x.Capabilities
	}
	return nil
}

type ExtensionCapability struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Version       int32                  `protobuf:"varint,2,opt,name=version,proto3" json:"version,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ExtensionCapability) Reset() {
	*x = ExtensionCapability{}
	mi := &file_daemon_extended_service_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ExtensionCapability) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExtensionCapability) ProtoMessage() {}

func (x *ExtensionCapability) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExtensionCapability.ProtoReflect.Descriptor instead.
func (*ExtensionCapability) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{1}
}

func (x *ExtensionCapability) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ExtensionCapability) GetVersion() int32 {
	if x != nil {
		return x.Version
	}
	return 0
}

// Force relay is process-wide: it applies to every Tailscale endpoint.
type SetTailscaleForceRelayRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Enabled       bool                   `protobuf:"varint,1,opt,name=enabled,proto3" json:"enabled,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SetTailscaleForceRelayRequest) Reset() {
	*x = SetTailscaleForceRelayRequest{}
	mi := &file_daemon_extended_service_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SetTailscaleForceRelayRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetTailscaleForceRelayRequest) ProtoMessage() {}

func (x *SetTailscaleForceRelayRequest) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SetTailscaleForceRelayRequest.ProtoReflect.Descriptor instead.
func (*SetTailscaleForceRelayRequest) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{2}
}

func (x *SetTailscaleForceRelayRequest) GetEnabled() bool {
	if x != nil {
		return x.Enabled
	}
	return false
}

type TailscaleForceRelayStatus struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Enabled       bool                   `protobuf:"varint,1,opt,name=enabled,proto3" json:"enabled,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *TailscaleForceRelayStatus) Reset() {
	*x = TailscaleForceRelayStatus{}
	mi := &file_daemon_extended_service_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *TailscaleForceRelayStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TailscaleForceRelayStatus) ProtoMessage() {}

func (x *TailscaleForceRelayStatus) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TailscaleForceRelayStatus.ProtoReflect.Descriptor instead.
func (*TailscaleForceRelayStatus) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{3}
}

func (x *TailscaleForceRelayStatus) GetEnabled() bool {
	if x != nil {
		return x.Enabled
	}
	return false
}

type OutboundProviderRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Tag           string                 `protobuf:"bytes,1,opt,name=tag,proto3" json:"tag,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OutboundProviderRequest) Reset() {
	*x = OutboundProviderRequest{}
	mi := &file_daemon_extended_service_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OutboundProviderRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OutboundProviderRequest) ProtoMessage() {}

func (x *OutboundProviderRequest) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OutboundProviderRequest.ProtoReflect.Descriptor instead.
func (*OutboundProviderRequest) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{4}
}

func (x *OutboundProviderRequest) GetTag() string {
	if x != nil {
		return x.Tag
	}
	return ""
}

type OutboundProviderStatusUpdate struct {
	state         protoimpl.MessageState    `protogen:"open.v1"`
	Providers     []*OutboundProviderStatus `protobuf:"bytes,1,rep,name=providers,proto3" json:"providers,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OutboundProviderStatusUpdate) Reset() {
	*x = OutboundProviderStatusUpdate{}
	mi := &file_daemon_extended_service_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OutboundProviderStatusUpdate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OutboundProviderStatusUpdate) ProtoMessage() {}

func (x *OutboundProviderStatusUpdate) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OutboundProviderStatusUpdate.ProtoReflect.Descriptor instead.
func (*OutboundProviderStatusUpdate) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{5}
}

func (x *OutboundProviderStatusUpdate) GetProviders() []*OutboundProviderStatus {
	if x != nil {
		return x.Providers
	}
	return nil
}

type OutboundProviderStatus struct {
	state            protoimpl.MessageState            `protogen:"open.v1"`
	Tag              string                            `protobuf:"bytes,1,opt,name=tag,proto3" json:"tag,omitempty"`
	Type             string                            `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	UpdatedAt        int64                             `protobuf:"varint,3,opt,name=updatedAt,proto3" json:"updatedAt,omitempty"`
	Updating         bool                              `protobuf:"varint,4,opt,name=updating,proto3" json:"updating,omitempty"`
	LastError        string                            `protobuf:"bytes,5,opt,name=lastError,proto3" json:"lastError,omitempty"`
	SubscriptionInfo *OutboundProviderSubscriptionInfo `protobuf:"bytes,6,opt,name=subscriptionInfo,proto3" json:"subscriptionInfo,omitempty"`
	Items            []*OutboundProviderItem           `protobuf:"bytes,7,rep,name=items,proto3" json:"items,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *OutboundProviderStatus) Reset() {
	*x = OutboundProviderStatus{}
	mi := &file_daemon_extended_service_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OutboundProviderStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OutboundProviderStatus) ProtoMessage() {}

func (x *OutboundProviderStatus) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OutboundProviderStatus.ProtoReflect.Descriptor instead.
func (*OutboundProviderStatus) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{6}
}

func (x *OutboundProviderStatus) GetTag() string {
	if x != nil {
		return x.Tag
	}
	return ""
}

func (x *OutboundProviderStatus) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *OutboundProviderStatus) GetUpdatedAt() int64 {
	if x != nil {
		return x.UpdatedAt
	}
	return 0
}

func (x *OutboundProviderStatus) GetUpdating() bool {
	if x != nil {
		return x.Updating
	}
	return false
}

func (x *OutboundProviderStatus) GetLastError() string {
	if x != nil {
		return x.LastError
	}
	return ""
}

func (x *OutboundProviderStatus) GetSubscriptionInfo() *OutboundProviderSubscriptionInfo {
	if x != nil {
		return x.SubscriptionInfo
	}
	return nil
}

func (x *OutboundProviderStatus) GetItems() []*OutboundProviderItem {
	if x != nil {
		return x.Items
	}
	return nil
}

type OutboundProviderSubscriptionInfo struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Upload        int64                  `protobuf:"varint,1,opt,name=upload,proto3" json:"upload,omitempty"`
	Download      int64                  `protobuf:"varint,2,opt,name=download,proto3" json:"download,omitempty"`
	Total         int64                  `protobuf:"varint,3,opt,name=total,proto3" json:"total,omitempty"`
	ExpireTime    int64                  `protobuf:"varint,4,opt,name=expireTime,proto3" json:"expireTime,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OutboundProviderSubscriptionInfo) Reset() {
	*x = OutboundProviderSubscriptionInfo{}
	mi := &file_daemon_extended_service_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OutboundProviderSubscriptionInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OutboundProviderSubscriptionInfo) ProtoMessage() {}

func (x *OutboundProviderSubscriptionInfo) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OutboundProviderSubscriptionInfo.ProtoReflect.Descriptor instead.
func (*OutboundProviderSubscriptionInfo) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{7}
}

func (x *OutboundProviderSubscriptionInfo) GetUpload() int64 {
	if x != nil {
		return x.Upload
	}
	return 0
}

func (x *OutboundProviderSubscriptionInfo) GetDownload() int64 {
	if x != nil {
		return x.Download
	}
	return 0
}

func (x *OutboundProviderSubscriptionInfo) GetTotal() int64 {
	if x != nil {
		return x.Total
	}
	return 0
}

func (x *OutboundProviderSubscriptionInfo) GetExpireTime() int64 {
	if x != nil {
		return x.ExpireTime
	}
	return 0
}

type OutboundProviderItem struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Tag           string                 `protobuf:"bytes,1,opt,name=tag,proto3" json:"tag,omitempty"`
	Type          string                 `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	UrlTestTime   int64                  `protobuf:"varint,3,opt,name=urlTestTime,proto3" json:"urlTestTime,omitempty"`
	UrlTestDelay  int32                  `protobuf:"varint,4,opt,name=urlTestDelay,proto3" json:"urlTestDelay,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OutboundProviderItem) Reset() {
	*x = OutboundProviderItem{}
	mi := &file_daemon_extended_service_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OutboundProviderItem) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OutboundProviderItem) ProtoMessage() {}

func (x *OutboundProviderItem) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OutboundProviderItem.ProtoReflect.Descriptor instead.
func (*OutboundProviderItem) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{8}
}

func (x *OutboundProviderItem) GetTag() string {
	if x != nil {
		return x.Tag
	}
	return ""
}

func (x *OutboundProviderItem) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *OutboundProviderItem) GetUrlTestTime() int64 {
	if x != nil {
		return x.UrlTestTime
	}
	return 0
}

func (x *OutboundProviderItem) GetUrlTestDelay() int32 {
	if x != nil {
		return x.UrlTestDelay
	}
	return 0
}

var File_daemon_extended_service_proto protoreflect.FileDescriptor

const file_daemon_extended_service_proto_rawDesc = "" +
	"\n" +
	"\x1ddaemon/extended_service.proto\x12\x06daemon\x1a\x1bgoogle/protobuf/empty.proto\"X\n" +
	"\x15ExtensionCapabilities\x12?\n" +
	"\fcapabilities\x18\x01 \x03(\v2\x1b.daemon.ExtensionCapabilityR\fcapabilities\"C\n" +
	"\x13ExtensionCapability\x12\x12\n" +
	"\x04name\x18\x01 \x01(\tR\x04name\x12\x18\n" +
	"\aversion\x18\x02 \x01(\x05R\aversion\"9\n" +
	"\x1dSetTailscaleForceRelayRequest\x12\x18\n" +
	"\aenabled\x18\x01 \x01(\bR\aenabled\"5\n" +
	"\x19TailscaleForceRelayStatus\x12\x18\n" +
	"\aenabled\x18\x01 \x01(\bR\aenabled\"+\n" +
	"\x17OutboundProviderRequest\x12\x10\n" +
	"\x03tag\x18\x01 \x01(\tR\x03tag\"\\\n" +
	"\x1cOutboundProviderStatusUpdate\x12<\n" +
	"\tproviders\x18\x01 \x03(\v2\x1e.daemon.OutboundProviderStatusR\tproviders\"\xa0\x02\n" +
	"\x16OutboundProviderStatus\x12\x10\n" +
	"\x03tag\x18\x01 \x01(\tR\x03tag\x12\x12\n" +
	"\x04type\x18\x02 \x01(\tR\x04type\x12\x1c\n" +
	"\tupdatedAt\x18\x03 \x01(\x03R\tupdatedAt\x12\x1a\n" +
	"\bupdating\x18\x04 \x01(\bR\bupdating\x12\x1c\n" +
	"\tlastError\x18\x05 \x01(\tR\tlastError\x12T\n" +
	"\x10subscriptionInfo\x18\x06 \x01(\v2(.daemon.OutboundProviderSubscriptionInfoR\x10subscriptionInfo\x122\n" +
	"\x05items\x18\a \x03(\v2\x1c.daemon.OutboundProviderItemR\x05items\"\x8c\x01\n" +
	" OutboundProviderSubscriptionInfo\x12\x16\n" +
	"\x06upload\x18\x01 \x01(\x03R\x06upload\x12\x1a\n" +
	"\bdownload\x18\x02 \x01(\x03R\bdownload\x12\x14\n" +
	"\x05total\x18\x03 \x01(\x03R\x05total\x12\x1e\n" +
	"\n" +
	"expireTime\x18\x04 \x01(\x03R\n" +
	"expireTime\"\x82\x01\n" +
	"\x14OutboundProviderItem\x12\x10\n" +
	"\x03tag\x18\x01 \x01(\tR\x03tag\x12\x12\n" +
	"\x04type\x18\x02 \x01(\tR\x04type\x12 \n" +
	"\vurlTestTime\x18\x03 \x01(\x03R\vurlTestTime\x12\"\n" +
	"\furlTestDelay\x18\x04 \x01(\x05R\furlTestDelay2\xd2\x03\n" +
	"\x0fExtendedService\x12J\n" +
	"\x0fGetCapabilities\x12\x16.google.protobuf.Empty\x1a\x1d.daemon.ExtensionCapabilities\"\x00\x12Y\n" +
	"\x16SetTailscaleForceRelay\x12%.daemon.SetTailscaleForceRelayRequest\x1a\x16.google.protobuf.Empty\"\x00\x12c\n" +
	"\"SubscribeTailscaleForceRelayStatus\x12\x16.google.protobuf.Empty\x1a!.daemon.TailscaleForceRelayStatus\"\x000\x01\x12^\n" +
	"\x1aSubscribeOutboundProviders\x12\x16.google.protobuf.Empty\x1a$.daemon.OutboundProviderStatusUpdate\"\x000\x01\x12S\n" +
	"\x16UpdateOutboundProvider\x12\x1f.daemon.OutboundProviderRequest\x1a\x16.google.protobuf.Empty\"\x00B%Z#github.com/sagernet/sing-box/daemonb\x06proto3"

var (
	file_daemon_extended_service_proto_rawDescOnce sync.Once
	file_daemon_extended_service_proto_rawDescData []byte
)

func file_daemon_extended_service_proto_rawDescGZIP() []byte {
	file_daemon_extended_service_proto_rawDescOnce.Do(func() {
		file_daemon_extended_service_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_daemon_extended_service_proto_rawDesc), len(file_daemon_extended_service_proto_rawDesc)))
	})
	return file_daemon_extended_service_proto_rawDescData
}

var (
	file_daemon_extended_service_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
	file_daemon_extended_service_proto_goTypes  = []any{
		(*ExtensionCapabilities)(nil),            // 0: daemon.ExtensionCapabilities
		(*ExtensionCapability)(nil),              // 1: daemon.ExtensionCapability
		(*SetTailscaleForceRelayRequest)(nil),    // 2: daemon.SetTailscaleForceRelayRequest
		(*TailscaleForceRelayStatus)(nil),        // 3: daemon.TailscaleForceRelayStatus
		(*OutboundProviderRequest)(nil),          // 4: daemon.OutboundProviderRequest
		(*OutboundProviderStatusUpdate)(nil),     // 5: daemon.OutboundProviderStatusUpdate
		(*OutboundProviderStatus)(nil),           // 6: daemon.OutboundProviderStatus
		(*OutboundProviderSubscriptionInfo)(nil), // 7: daemon.OutboundProviderSubscriptionInfo
		(*OutboundProviderItem)(nil),             // 8: daemon.OutboundProviderItem
		(*emptypb.Empty)(nil),                    // 9: google.protobuf.Empty
	}
)

var file_daemon_extended_service_proto_depIdxs = []int32{
	1, // 0: daemon.ExtensionCapabilities.capabilities:type_name -> daemon.ExtensionCapability
	6, // 1: daemon.OutboundProviderStatusUpdate.providers:type_name -> daemon.OutboundProviderStatus
	7, // 2: daemon.OutboundProviderStatus.subscriptionInfo:type_name -> daemon.OutboundProviderSubscriptionInfo
	8, // 3: daemon.OutboundProviderStatus.items:type_name -> daemon.OutboundProviderItem
	9, // 4: daemon.ExtendedService.GetCapabilities:input_type -> google.protobuf.Empty
	2, // 5: daemon.ExtendedService.SetTailscaleForceRelay:input_type -> daemon.SetTailscaleForceRelayRequest
	9, // 6: daemon.ExtendedService.SubscribeTailscaleForceRelayStatus:input_type -> google.protobuf.Empty
	9, // 7: daemon.ExtendedService.SubscribeOutboundProviders:input_type -> google.protobuf.Empty
	4, // 8: daemon.ExtendedService.UpdateOutboundProvider:input_type -> daemon.OutboundProviderRequest
	0, // 9: daemon.ExtendedService.GetCapabilities:output_type -> daemon.ExtensionCapabilities
	9, // 10: daemon.ExtendedService.SetTailscaleForceRelay:output_type -> google.protobuf.Empty
	3, // 11: daemon.ExtendedService.SubscribeTailscaleForceRelayStatus:output_type -> daemon.TailscaleForceRelayStatus
	5, // 12: daemon.ExtendedService.SubscribeOutboundProviders:output_type -> daemon.OutboundProviderStatusUpdate
	9, // 13: daemon.ExtendedService.UpdateOutboundProvider:output_type -> google.protobuf.Empty
	9, // [9:14] is the sub-list for method output_type
	4, // [4:9] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_daemon_extended_service_proto_init() }
func file_daemon_extended_service_proto_init() {
	if File_daemon_extended_service_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_daemon_extended_service_proto_rawDesc), len(file_daemon_extended_service_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_daemon_extended_service_proto_goTypes,
		DependencyIndexes: file_daemon_extended_service_proto_depIdxs,
		MessageInfos:      file_daemon_extended_service_proto_msgTypes,
	}.Build()
	File_daemon_extended_service_proto = out.File
	file_daemon_extended_service_proto_goTypes = nil
	file_daemon_extended_service_proto_depIdxs = nil
}
