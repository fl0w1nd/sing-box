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

type DNSQueryLogStatus struct {
	state      protoimpl.MessageState `protogen:"open.v1"`
	Enabled    bool                   `protobuf:"varint,1,opt,name=enabled,proto3" json:"enabled,omitempty"`
	Persistent bool                   `protobuf:"varint,2,opt,name=persistent,proto3" json:"persistent,omitempty"`
	Path       string                 `protobuf:"bytes,3,opt,name=path,proto3" json:"path,omitempty"`
	// Seconds.
	Retention      int64 `protobuf:"varint,4,opt,name=retention,proto3" json:"retention,omitempty"`
	MaxSize        int64 `protobuf:"varint,5,opt,name=maxSize,proto3" json:"maxSize,omitempty"`
	DiskUsage      int64 `protobuf:"varint,6,opt,name=diskUsage,proto3" json:"diskUsage,omitempty"`
	Segments       int32 `protobuf:"varint,7,opt,name=segments,proto3" json:"segments,omitempty"`
	MemoryEntries  int32 `protobuf:"varint,8,opt,name=memoryEntries,proto3" json:"memoryEntries,omitempty"`
	PendingEntries int32 `protobuf:"varint,9,opt,name=pendingEntries,proto3" json:"pendingEntries,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *DNSQueryLogStatus) Reset() {
	*x = DNSQueryLogStatus{}
	mi := &file_daemon_extended_service_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DNSQueryLogStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DNSQueryLogStatus) ProtoMessage() {}

func (x *DNSQueryLogStatus) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DNSQueryLogStatus.ProtoReflect.Descriptor instead.
func (*DNSQueryLogStatus) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{9}
}

func (x *DNSQueryLogStatus) GetEnabled() bool {
	if x != nil {
		return x.Enabled
	}
	return false
}

func (x *DNSQueryLogStatus) GetPersistent() bool {
	if x != nil {
		return x.Persistent
	}
	return false
}

func (x *DNSQueryLogStatus) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *DNSQueryLogStatus) GetRetention() int64 {
	if x != nil {
		return x.Retention
	}
	return 0
}

func (x *DNSQueryLogStatus) GetMaxSize() int64 {
	if x != nil {
		return x.MaxSize
	}
	return 0
}

func (x *DNSQueryLogStatus) GetDiskUsage() int64 {
	if x != nil {
		return x.DiskUsage
	}
	return 0
}

func (x *DNSQueryLogStatus) GetSegments() int32 {
	if x != nil {
		return x.Segments
	}
	return 0
}

func (x *DNSQueryLogStatus) GetMemoryEntries() int32 {
	if x != nil {
		return x.MemoryEntries
	}
	return 0
}

func (x *DNSQueryLogStatus) GetPendingEntries() int32 {
	if x != nil {
		return x.PendingEntries
	}
	return 0
}

// Empty fields match everything. Queries return entries newest first;
// subscriptions ignore before and limit.
type DNSQueryLogRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Matches the domain, source, process or answers, case-insensitive.
	Search    string `protobuf:"bytes,1,opt,name=search,proto3" json:"search,omitempty"`
	QueryType string `protobuf:"bytes,2,opt,name=queryType,proto3" json:"queryType,omitempty"`
	Status    string `protobuf:"bytes,3,opt,name=status,proto3" json:"status,omitempty"`
	Server    string `protobuf:"bytes,4,opt,name=server,proto3" json:"server,omitempty"`
	Inbound   string `protobuf:"bytes,5,opt,name=inbound,proto3" json:"inbound,omitempty"`
	// Unix milliseconds.
	Since int64 `protobuf:"varint,6,opt,name=since,proto3" json:"since,omitempty"`
	Until int64 `protobuf:"varint,7,opt,name=until,proto3" json:"until,omitempty"`
	// Returns entries with an ID below this cursor when not zero.
	Before        uint64 `protobuf:"varint,8,opt,name=before,proto3" json:"before,omitempty"`
	Limit         int32  `protobuf:"varint,9,opt,name=limit,proto3" json:"limit,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DNSQueryLogRequest) Reset() {
	*x = DNSQueryLogRequest{}
	mi := &file_daemon_extended_service_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DNSQueryLogRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DNSQueryLogRequest) ProtoMessage() {}

func (x *DNSQueryLogRequest) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DNSQueryLogRequest.ProtoReflect.Descriptor instead.
func (*DNSQueryLogRequest) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{10}
}

func (x *DNSQueryLogRequest) GetSearch() string {
	if x != nil {
		return x.Search
	}
	return ""
}

func (x *DNSQueryLogRequest) GetQueryType() string {
	if x != nil {
		return x.QueryType
	}
	return ""
}

func (x *DNSQueryLogRequest) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

func (x *DNSQueryLogRequest) GetServer() string {
	if x != nil {
		return x.Server
	}
	return ""
}

func (x *DNSQueryLogRequest) GetInbound() string {
	if x != nil {
		return x.Inbound
	}
	return ""
}

func (x *DNSQueryLogRequest) GetSince() int64 {
	if x != nil {
		return x.Since
	}
	return 0
}

func (x *DNSQueryLogRequest) GetUntil() int64 {
	if x != nil {
		return x.Until
	}
	return 0
}

func (x *DNSQueryLogRequest) GetBefore() uint64 {
	if x != nil {
		return x.Before
	}
	return 0
}

func (x *DNSQueryLogRequest) GetLimit() int32 {
	if x != nil {
		return x.Limit
	}
	return 0
}

type DNSQueryLogPage struct {
	state   protoimpl.MessageState `protogen:"open.v1"`
	Entries []*DNSQueryLogEntry    `protobuf:"bytes,1,rep,name=entries,proto3" json:"entries,omitempty"`
	// Cursor for the next page, or zero when there is none.
	Next          uint64 `protobuf:"varint,2,opt,name=next,proto3" json:"next,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DNSQueryLogPage) Reset() {
	*x = DNSQueryLogPage{}
	mi := &file_daemon_extended_service_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DNSQueryLogPage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DNSQueryLogPage) ProtoMessage() {}

func (x *DNSQueryLogPage) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DNSQueryLogPage.ProtoReflect.Descriptor instead.
func (*DNSQueryLogPage) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{11}
}

func (x *DNSQueryLogPage) GetEntries() []*DNSQueryLogEntry {
	if x != nil {
		return x.Entries
	}
	return nil
}

func (x *DNSQueryLogPage) GetNext() uint64 {
	if x != nil {
		return x.Next
	}
	return 0
}

type DNSQueryLogEntry struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	Id    uint64                 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	// Unix milliseconds.
	Time int64 `protobuf:"varint,2,opt,name=time,proto3" json:"time,omitempty"`
	// Microseconds.
	Duration      int64                  `protobuf:"varint,3,opt,name=duration,proto3" json:"duration,omitempty"`
	Method        string                 `protobuf:"bytes,4,opt,name=method,proto3" json:"method,omitempty"`
	Domain        string                 `protobuf:"bytes,5,opt,name=domain,proto3" json:"domain,omitempty"`
	QueryType     string                 `protobuf:"bytes,6,opt,name=queryType,proto3" json:"queryType,omitempty"`
	Inbound       string                 `protobuf:"bytes,7,opt,name=inbound,proto3" json:"inbound,omitempty"`
	InboundType   string                 `protobuf:"bytes,8,opt,name=inboundType,proto3" json:"inboundType,omitempty"`
	Source        string                 `protobuf:"bytes,9,opt,name=source,proto3" json:"source,omitempty"`
	User          string                 `protobuf:"bytes,10,opt,name=user,proto3" json:"user,omitempty"`
	Process       string                 `protobuf:"bytes,11,opt,name=process,proto3" json:"process,omitempty"`
	Outbound      string                 `protobuf:"bytes,12,opt,name=outbound,proto3" json:"outbound,omitempty"`
	Destination   string                 `protobuf:"bytes,13,opt,name=destination,proto3" json:"destination,omitempty"`
	Rules         []*DNSQueryLogRule     `protobuf:"bytes,14,rep,name=rules,proto3" json:"rules,omitempty"`
	Upstreams     []*DNSQueryLogUpstream `protobuf:"bytes,15,rep,name=upstreams,proto3" json:"upstreams,omitempty"`
	Cached        bool                   `protobuf:"varint,16,opt,name=cached,proto3" json:"cached,omitempty"`
	Optimistic    bool                   `protobuf:"varint,17,opt,name=optimistic,proto3" json:"optimistic,omitempty"`
	Rejected      bool                   `protobuf:"varint,18,opt,name=rejected,proto3" json:"rejected,omitempty"`
	Status        string                 `protobuf:"bytes,19,opt,name=status,proto3" json:"status,omitempty"`
	Rcode         string                 `protobuf:"bytes,20,opt,name=rcode,proto3" json:"rcode,omitempty"`
	Answers       []string               `protobuf:"bytes,21,rep,name=answers,proto3" json:"answers,omitempty"`
	Ttl           uint32                 `protobuf:"varint,22,opt,name=ttl,proto3" json:"ttl,omitempty"`
	Error         string                 `protobuf:"bytes,23,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DNSQueryLogEntry) Reset() {
	*x = DNSQueryLogEntry{}
	mi := &file_daemon_extended_service_proto_msgTypes[12]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DNSQueryLogEntry) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DNSQueryLogEntry) ProtoMessage() {}

func (x *DNSQueryLogEntry) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[12]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DNSQueryLogEntry.ProtoReflect.Descriptor instead.
func (*DNSQueryLogEntry) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{12}
}

func (x *DNSQueryLogEntry) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *DNSQueryLogEntry) GetTime() int64 {
	if x != nil {
		return x.Time
	}
	return 0
}

func (x *DNSQueryLogEntry) GetDuration() int64 {
	if x != nil {
		return x.Duration
	}
	return 0
}

func (x *DNSQueryLogEntry) GetMethod() string {
	if x != nil {
		return x.Method
	}
	return ""
}

func (x *DNSQueryLogEntry) GetDomain() string {
	if x != nil {
		return x.Domain
	}
	return ""
}

func (x *DNSQueryLogEntry) GetQueryType() string {
	if x != nil {
		return x.QueryType
	}
	return ""
}

func (x *DNSQueryLogEntry) GetInbound() string {
	if x != nil {
		return x.Inbound
	}
	return ""
}

func (x *DNSQueryLogEntry) GetInboundType() string {
	if x != nil {
		return x.InboundType
	}
	return ""
}

func (x *DNSQueryLogEntry) GetSource() string {
	if x != nil {
		return x.Source
	}
	return ""
}

func (x *DNSQueryLogEntry) GetUser() string {
	if x != nil {
		return x.User
	}
	return ""
}

func (x *DNSQueryLogEntry) GetProcess() string {
	if x != nil {
		return x.Process
	}
	return ""
}

func (x *DNSQueryLogEntry) GetOutbound() string {
	if x != nil {
		return x.Outbound
	}
	return ""
}

func (x *DNSQueryLogEntry) GetDestination() string {
	if x != nil {
		return x.Destination
	}
	return ""
}

func (x *DNSQueryLogEntry) GetRules() []*DNSQueryLogRule {
	if x != nil {
		return x.Rules
	}
	return nil
}

func (x *DNSQueryLogEntry) GetUpstreams() []*DNSQueryLogUpstream {
	if x != nil {
		return x.Upstreams
	}
	return nil
}

func (x *DNSQueryLogEntry) GetCached() bool {
	if x != nil {
		return x.Cached
	}
	return false
}

func (x *DNSQueryLogEntry) GetOptimistic() bool {
	if x != nil {
		return x.Optimistic
	}
	return false
}

func (x *DNSQueryLogEntry) GetRejected() bool {
	if x != nil {
		return x.Rejected
	}
	return false
}

func (x *DNSQueryLogEntry) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

func (x *DNSQueryLogEntry) GetRcode() string {
	if x != nil {
		return x.Rcode
	}
	return ""
}

func (x *DNSQueryLogEntry) GetAnswers() []string {
	if x != nil {
		return x.Answers
	}
	return nil
}

func (x *DNSQueryLogEntry) GetTtl() uint32 {
	if x != nil {
		return x.Ttl
	}
	return 0
}

func (x *DNSQueryLogEntry) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

type DNSQueryLogRule struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Index         int32                  `protobuf:"varint,1,opt,name=index,proto3" json:"index,omitempty"`
	Rule          string                 `protobuf:"bytes,2,opt,name=rule,proto3" json:"rule,omitempty"`
	Action        string                 `protobuf:"bytes,3,opt,name=action,proto3" json:"action,omitempty"`
	ActionType    string                 `protobuf:"bytes,4,opt,name=actionType,proto3" json:"actionType,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DNSQueryLogRule) Reset() {
	*x = DNSQueryLogRule{}
	mi := &file_daemon_extended_service_proto_msgTypes[13]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DNSQueryLogRule) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DNSQueryLogRule) ProtoMessage() {}

func (x *DNSQueryLogRule) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[13]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DNSQueryLogRule.ProtoReflect.Descriptor instead.
func (*DNSQueryLogRule) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{13}
}

func (x *DNSQueryLogRule) GetIndex() int32 {
	if x != nil {
		return x.Index
	}
	return 0
}

func (x *DNSQueryLogRule) GetRule() string {
	if x != nil {
		return x.Rule
	}
	return ""
}

func (x *DNSQueryLogRule) GetAction() string {
	if x != nil {
		return x.Action
	}
	return ""
}

func (x *DNSQueryLogRule) GetActionType() string {
	if x != nil {
		return x.ActionType
	}
	return ""
}

type DNSQueryLogUpstream struct {
	state     protoimpl.MessageState `protogen:"open.v1"`
	Server    string                 `protobuf:"bytes,1,opt,name=server,proto3" json:"server,omitempty"`
	Type      string                 `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	QueryType string                 `protobuf:"bytes,3,opt,name=queryType,proto3" json:"queryType,omitempty"`
	// Microseconds.
	Duration int64    `protobuf:"varint,4,opt,name=duration,proto3" json:"duration,omitempty"`
	Rcode    string   `protobuf:"bytes,5,opt,name=rcode,proto3" json:"rcode,omitempty"`
	Error    string   `protobuf:"bytes,6,opt,name=error,proto3" json:"error,omitempty"`
	Rejected bool     `protobuf:"varint,7,opt,name=rejected,proto3" json:"rejected,omitempty"`
	Answers  []string `protobuf:"bytes,8,rep,name=answers,proto3" json:"answers,omitempty"`
	// The configured address, such as https://dns.google/dns-query, or the
	// addresses a local or dhcp server used.
	Address string `protobuf:"bytes,9,opt,name=address,proto3" json:"address,omitempty"`
	// The configured detour, or the endpoint of a tailscale server.
	Detour        string `protobuf:"bytes,10,opt,name=detour,proto3" json:"detour,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DNSQueryLogUpstream) Reset() {
	*x = DNSQueryLogUpstream{}
	mi := &file_daemon_extended_service_proto_msgTypes[14]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DNSQueryLogUpstream) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DNSQueryLogUpstream) ProtoMessage() {}

func (x *DNSQueryLogUpstream) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[14]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DNSQueryLogUpstream.ProtoReflect.Descriptor instead.
func (*DNSQueryLogUpstream) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{14}
}

func (x *DNSQueryLogUpstream) GetServer() string {
	if x != nil {
		return x.Server
	}
	return ""
}

func (x *DNSQueryLogUpstream) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *DNSQueryLogUpstream) GetQueryType() string {
	if x != nil {
		return x.QueryType
	}
	return ""
}

func (x *DNSQueryLogUpstream) GetDuration() int64 {
	if x != nil {
		return x.Duration
	}
	return 0
}

func (x *DNSQueryLogUpstream) GetRcode() string {
	if x != nil {
		return x.Rcode
	}
	return ""
}

func (x *DNSQueryLogUpstream) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

func (x *DNSQueryLogUpstream) GetRejected() bool {
	if x != nil {
		return x.Rejected
	}
	return false
}

func (x *DNSQueryLogUpstream) GetAnswers() []string {
	if x != nil {
		return x.Answers
	}
	return nil
}

func (x *DNSQueryLogUpstream) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *DNSQueryLogUpstream) GetDetour() string {
	if x != nil {
		return x.Detour
	}
	return ""
}

type DNSQueryLogStatsRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Selects the entries as in queries; before and limit are ignored.
	Filter *DNSQueryLogRequest `protobuf:"bytes,1,opt,name=filter,proto3" json:"filter,omitempty"`
	// Seconds; zero picks one hour. It grows when the range would need more
	// than 1000 buckets.
	Bucket int64 `protobuf:"varint,2,opt,name=bucket,proto3" json:"bucket,omitempty"`
	// Entries in each ranking; zero picks 10, at most 100.
	Top           int32 `protobuf:"varint,3,opt,name=top,proto3" json:"top,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DNSQueryLogStatsRequest) Reset() {
	*x = DNSQueryLogStatsRequest{}
	mi := &file_daemon_extended_service_proto_msgTypes[15]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DNSQueryLogStatsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DNSQueryLogStatsRequest) ProtoMessage() {}

func (x *DNSQueryLogStatsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[15]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DNSQueryLogStatsRequest.ProtoReflect.Descriptor instead.
func (*DNSQueryLogStatsRequest) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{15}
}

func (x *DNSQueryLogStatsRequest) GetFilter() *DNSQueryLogRequest {
	if x != nil {
		return x.Filter
	}
	return nil
}

func (x *DNSQueryLogStatsRequest) GetBucket() int64 {
	if x != nil {
		return x.Bucket
	}
	return 0
}

func (x *DNSQueryLogStatsRequest) GetTop() int32 {
	if x != nil {
		return x.Top
	}
	return 0
}

type DNSQueryLogStats struct {
	state    protoimpl.MessageState `protogen:"open.v1"`
	Total    int64                  `protobuf:"varint,1,opt,name=total,proto3" json:"total,omitempty"`
	Statuses map[string]int64       `protobuf:"bytes,2,rep,name=statuses,proto3" json:"statuses,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"varint,2,opt,name=value"`
	// Seconds, the bucket length used.
	Bucket int64 `protobuf:"varint,3,opt,name=bucket,proto3" json:"bucket,omitempty"`
	// From since, or the oldest entry, to until, or now, empty buckets
	// included. Buckets start at a multiple of the bucket length in Unix time.
	Buckets    []*DNSQueryLogStatsBucket `protobuf:"bytes,4,rep,name=buckets,proto3" json:"buckets,omitempty"`
	TopDomains []*DNSQueryLogStatsCount  `protobuf:"bytes,5,rep,name=topDomains,proto3" json:"topDomains,omitempty"`
	// Domains answered by a reject or predefined rule.
	TopBlockedDomains []*DNSQueryLogStatsCount `protobuf:"bytes,6,rep,name=topBlockedDomains,proto3" json:"topBlockedDomains,omitempty"`
	// Source addresses without the port; entries without a source, such as
	// background refreshes, are left out.
	TopClients []*DNSQueryLogStatsCount `protobuf:"bytes,7,rep,name=topClients,proto3" json:"topClients,omitempty"`
	// Every query type of an entry counts, so a lookup of A and AAAA counts
	// for both.
	QueryTypes []*DNSQueryLogStatsCount    `protobuf:"bytes,8,rep,name=queryTypes,proto3" json:"queryTypes,omitempty"`
	Upstreams  []*DNSQueryLogUpstreamStats `protobuf:"bytes,9,rep,name=upstreams,proto3" json:"upstreams,omitempty"`
	// Of all entries.
	Duration *DNSQueryLogDuration `protobuf:"bytes,10,opt,name=duration,proto3" json:"duration,omitempty"`
	// Of entries with the upstream status.
	UpstreamDuration *DNSQueryLogDuration `protobuf:"bytes,11,opt,name=upstreamDuration,proto3" json:"upstreamDuration,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *DNSQueryLogStats) Reset() {
	*x = DNSQueryLogStats{}
	mi := &file_daemon_extended_service_proto_msgTypes[16]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DNSQueryLogStats) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DNSQueryLogStats) ProtoMessage() {}

func (x *DNSQueryLogStats) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[16]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DNSQueryLogStats.ProtoReflect.Descriptor instead.
func (*DNSQueryLogStats) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{16}
}

func (x *DNSQueryLogStats) GetTotal() int64 {
	if x != nil {
		return x.Total
	}
	return 0
}

func (x *DNSQueryLogStats) GetStatuses() map[string]int64 {
	if x != nil {
		return x.Statuses
	}
	return nil
}

func (x *DNSQueryLogStats) GetBucket() int64 {
	if x != nil {
		return x.Bucket
	}
	return 0
}

func (x *DNSQueryLogStats) GetBuckets() []*DNSQueryLogStatsBucket {
	if x != nil {
		return x.Buckets
	}
	return nil
}

func (x *DNSQueryLogStats) GetTopDomains() []*DNSQueryLogStatsCount {
	if x != nil {
		return x.TopDomains
	}
	return nil
}

func (x *DNSQueryLogStats) GetTopBlockedDomains() []*DNSQueryLogStatsCount {
	if x != nil {
		return x.TopBlockedDomains
	}
	return nil
}

func (x *DNSQueryLogStats) GetTopClients() []*DNSQueryLogStatsCount {
	if x != nil {
		return x.TopClients
	}
	return nil
}

func (x *DNSQueryLogStats) GetQueryTypes() []*DNSQueryLogStatsCount {
	if x != nil {
		return x.QueryTypes
	}
	return nil
}

func (x *DNSQueryLogStats) GetUpstreams() []*DNSQueryLogUpstreamStats {
	if x != nil {
		return x.Upstreams
	}
	return nil
}

func (x *DNSQueryLogStats) GetDuration() *DNSQueryLogDuration {
	if x != nil {
		return x.Duration
	}
	return nil
}

func (x *DNSQueryLogStats) GetUpstreamDuration() *DNSQueryLogDuration {
	if x != nil {
		return x.UpstreamDuration
	}
	return nil
}

type DNSQueryLogStatsBucket struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Unix milliseconds, the start of the bucket.
	Time          int64            `protobuf:"varint,1,opt,name=time,proto3" json:"time,omitempty"`
	Total         int64            `protobuf:"varint,2,opt,name=total,proto3" json:"total,omitempty"`
	Statuses      map[string]int64 `protobuf:"bytes,3,rep,name=statuses,proto3" json:"statuses,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"varint,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DNSQueryLogStatsBucket) Reset() {
	*x = DNSQueryLogStatsBucket{}
	mi := &file_daemon_extended_service_proto_msgTypes[17]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DNSQueryLogStatsBucket) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DNSQueryLogStatsBucket) ProtoMessage() {}

func (x *DNSQueryLogStatsBucket) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[17]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DNSQueryLogStatsBucket.ProtoReflect.Descriptor instead.
func (*DNSQueryLogStatsBucket) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{17}
}

func (x *DNSQueryLogStatsBucket) GetTime() int64 {
	if x != nil {
		return x.Time
	}
	return 0
}

func (x *DNSQueryLogStatsBucket) GetTotal() int64 {
	if x != nil {
		return x.Total
	}
	return 0
}

func (x *DNSQueryLogStatsBucket) GetStatuses() map[string]int64 {
	if x != nil {
		return x.Statuses
	}
	return nil
}

type DNSQueryLogStatsCount struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Key           string                 `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Count         int64                  `protobuf:"varint,2,opt,name=count,proto3" json:"count,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DNSQueryLogStatsCount) Reset() {
	*x = DNSQueryLogStatsCount{}
	mi := &file_daemon_extended_service_proto_msgTypes[18]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DNSQueryLogStatsCount) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DNSQueryLogStatsCount) ProtoMessage() {}

func (x *DNSQueryLogStatsCount) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[18]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DNSQueryLogStatsCount.ProtoReflect.Descriptor instead.
func (*DNSQueryLogStatsCount) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{18}
}

func (x *DNSQueryLogStatsCount) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *DNSQueryLogStatsCount) GetCount() int64 {
	if x != nil {
		return x.Count
	}
	return 0
}

type DNSQueryLogUpstreamStats struct {
	state     protoimpl.MessageState `protogen:"open.v1"`
	Server    string                 `protobuf:"bytes,1,opt,name=server,proto3" json:"server,omitempty"`
	Type      string                 `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	Exchanges int64                  `protobuf:"varint,3,opt,name=exchanges,proto3" json:"exchanges,omitempty"`
	Failed    int64                  `protobuf:"varint,4,opt,name=failed,proto3" json:"failed,omitempty"`
	Rejected  int64                  `protobuf:"varint,5,opt,name=rejected,proto3" json:"rejected,omitempty"`
	// Of the exchanges that did not fail.
	Duration *DNSQueryLogDuration `protobuf:"bytes,6,opt,name=duration,proto3" json:"duration,omitempty"`
	// Of the newest exchange.
	Address       string `protobuf:"bytes,7,opt,name=address,proto3" json:"address,omitempty"`
	Detour        string `protobuf:"bytes,8,opt,name=detour,proto3" json:"detour,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DNSQueryLogUpstreamStats) Reset() {
	*x = DNSQueryLogUpstreamStats{}
	mi := &file_daemon_extended_service_proto_msgTypes[19]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DNSQueryLogUpstreamStats) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DNSQueryLogUpstreamStats) ProtoMessage() {}

func (x *DNSQueryLogUpstreamStats) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[19]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DNSQueryLogUpstreamStats.ProtoReflect.Descriptor instead.
func (*DNSQueryLogUpstreamStats) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{19}
}

func (x *DNSQueryLogUpstreamStats) GetServer() string {
	if x != nil {
		return x.Server
	}
	return ""
}

func (x *DNSQueryLogUpstreamStats) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *DNSQueryLogUpstreamStats) GetExchanges() int64 {
	if x != nil {
		return x.Exchanges
	}
	return 0
}

func (x *DNSQueryLogUpstreamStats) GetFailed() int64 {
	if x != nil {
		return x.Failed
	}
	return 0
}

func (x *DNSQueryLogUpstreamStats) GetRejected() int64 {
	if x != nil {
		return x.Rejected
	}
	return 0
}

func (x *DNSQueryLogUpstreamStats) GetDuration() *DNSQueryLogDuration {
	if x != nil {
		return x.Duration
	}
	return nil
}

func (x *DNSQueryLogUpstreamStats) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *DNSQueryLogUpstreamStats) GetDetour() string {
	if x != nil {
		return x.Detour
	}
	return ""
}

// Microseconds; p95 is the 95th percentile by the nearest rank.
type DNSQueryLogDuration struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Average       int64                  `protobuf:"varint,1,opt,name=average,proto3" json:"average,omitempty"`
	P95           int64                  `protobuf:"varint,2,opt,name=p95,proto3" json:"p95,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DNSQueryLogDuration) Reset() {
	*x = DNSQueryLogDuration{}
	mi := &file_daemon_extended_service_proto_msgTypes[20]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DNSQueryLogDuration) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DNSQueryLogDuration) ProtoMessage() {}

func (x *DNSQueryLogDuration) ProtoReflect() protoreflect.Message {
	mi := &file_daemon_extended_service_proto_msgTypes[20]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DNSQueryLogDuration.ProtoReflect.Descriptor instead.
func (*DNSQueryLogDuration) Descriptor() ([]byte, []int) {
	return file_daemon_extended_service_proto_rawDescGZIP(), []int{20}
}

func (x *DNSQueryLogDuration) GetAverage() int64 {
	if x != nil {
		return x.Average
	}
	return 0
}

func (x *DNSQueryLogDuration) GetP95() int64 {
	if x != nil {
		return x.P95
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
	"\furlTestDelay\x18\x04 \x01(\x05R\furlTestDelay\"\xa1\x02\n" +
	"\x11DNSQueryLogStatus\x12\x18\n" +
	"\aenabled\x18\x01 \x01(\bR\aenabled\x12\x1e\n" +
	"\n" +
	"persistent\x18\x02 \x01(\bR\n" +
	"persistent\x12\x12\n" +
	"\x04path\x18\x03 \x01(\tR\x04path\x12\x1c\n" +
	"\tretention\x18\x04 \x01(\x03R\tretention\x12\x18\n" +
	"\amaxSize\x18\x05 \x01(\x03R\amaxSize\x12\x1c\n" +
	"\tdiskUsage\x18\x06 \x01(\x03R\tdiskUsage\x12\x1a\n" +
	"\bsegments\x18\a \x01(\x05R\bsegments\x12$\n" +
	"\rmemoryEntries\x18\b \x01(\x05R\rmemoryEntries\x12&\n" +
	"\x0ependingEntries\x18\t \x01(\x05R\x0ependingEntries\"\xee\x01\n" +
	"\x12DNSQueryLogRequest\x12\x16\n" +
	"\x06search\x18\x01 \x01(\tR\x06search\x12\x1c\n" +
	"\tqueryType\x18\x02 \x01(\tR\tqueryType\x12\x16\n" +
	"\x06status\x18\x03 \x01(\tR\x06status\x12\x16\n" +
	"\x06server\x18\x04 \x01(\tR\x06server\x12\x18\n" +
	"\ainbound\x18\x05 \x01(\tR\ainbound\x12\x14\n" +
	"\x05since\x18\x06 \x01(\x03R\x05since\x12\x14\n" +
	"\x05until\x18\a \x01(\x03R\x05until\x12\x16\n" +
	"\x06before\x18\b \x01(\x04R\x06before\x12\x14\n" +
	"\x05limit\x18\t \x01(\x05R\x05limit\"Y\n" +
	"\x0fDNSQueryLogPage\x122\n" +
	"\aentries\x18\x01 \x03(\v2\x18.daemon.DNSQueryLogEntryR\aentries\x12\x12\n" +
	"\x04next\x18\x02 \x01(\x04R\x04next\"\x8e\x05\n" +
	"\x10DNSQueryLogEntry\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\x04R\x02id\x12\x12\n" +
	"\x04time\x18\x02 \x01(\x03R\x04time\x12\x1a\n" +
	"\bduration\x18\x03 \x01(\x03R\bduration\x12\x16\n" +
	"\x06method\x18\x04 \x01(\tR\x06method\x12\x16\n" +
	"\x06domain\x18\x05 \x01(\tR\x06domain\x12\x1c\n" +
	"\tqueryType\x18\x06 \x01(\tR\tqueryType\x12\x18\n" +
	"\ainbound\x18\a \x01(\tR\ainbound\x12 \n" +
	"\vinboundType\x18\b \x01(\tR\vinboundType\x12\x16\n" +
	"\x06source\x18\t \x01(\tR\x06source\x12\x12\n" +
	"\x04user\x18\n" +
	" \x01(\tR\x04user\x12\x18\n" +
	"\aprocess\x18\v \x01(\tR\aprocess\x12\x1a\n" +
	"\boutbound\x18\f \x01(\tR\boutbound\x12 \n" +
	"\vdestination\x18\r \x01(\tR\vdestination\x12-\n" +
	"\x05rules\x18\x0e \x03(\v2\x17.daemon.DNSQueryLogRuleR\x05rules\x129\n" +
	"\tupstreams\x18\x0f \x03(\v2\x1b.daemon.DNSQueryLogUpstreamR\tupstreams\x12\x16\n" +
	"\x06cached\x18\x10 \x01(\bR\x06cached\x12\x1e\n" +
	"\n" +
	"optimistic\x18\x11 \x01(\bR\n" +
	"optimistic\x12\x1a\n" +
	"\brejected\x18\x12 \x01(\bR\brejected\x12\x16\n" +
	"\x06status\x18\x13 \x01(\tR\x06status\x12\x14\n" +
	"\x05rcode\x18\x14 \x01(\tR\x05rcode\x12\x18\n" +
	"\aanswers\x18\x15 \x03(\tR\aanswers\x12\x10\n" +
	"\x03ttl\x18\x16 \x01(\rR\x03ttl\x12\x14\n" +
	"\x05error\x18\x17 \x01(\tR\x05error\"s\n" +
	"\x0fDNSQueryLogRule\x12\x14\n" +
	"\x05index\x18\x01 \x01(\x05R\x05index\x12\x12\n" +
	"\x04rule\x18\x02 \x01(\tR\x04rule\x12\x16\n" +
	"\x06action\x18\x03 \x01(\tR\x06action\x12\x1e\n" +
	"\n" +
	"actionType\x18\x04 \x01(\tR\n" +
	"actionType\"\x8f\x02\n" +
	"\x13DNSQueryLogUpstream\x12\x16\n" +
	"\x06server\x18\x01 \x01(\tR\x06server\x12\x12\n" +
	"\x04type\x18\x02 \x01(\tR\x04type\x12\x1c\n" +
	"\tqueryType\x18\x03 \x01(\tR\tqueryType\x12\x1a\n" +
	"\bduration\x18\x04 \x01(\x03R\bduration\x12\x14\n" +
	"\x05rcode\x18\x05 \x01(\tR\x05rcode\x12\x14\n" +
	"\x05error\x18\x06 \x01(\tR\x05error\x12\x1a\n" +
	"\brejected\x18\a \x01(\bR\brejected\x12\x18\n" +
	"\aanswers\x18\b \x03(\tR\aanswers\x12\x18\n" +
	"\aaddress\x18\t \x01(\tR\aaddress\x12\x16\n" +
	"\x06detour\x18\n" +
	" \x01(\tR\x06detour\"w\n" +
	"\x17DNSQueryLogStatsRequest\x122\n" +
	"\x06filter\x18\x01 \x01(\v2\x1a.daemon.DNSQueryLogRequestR\x06filter\x12\x16\n" +
	"\x06bucket\x18\x02 \x01(\x03R\x06bucket\x12\x10\n" +
	"\x03top\x18\x03 \x01(\x05R\x03top\"\xc7\x05\n" +
	"\x10DNSQueryLogStats\x12\x14\n" +
	"\x05total\x18\x01 \x01(\x03R\x05total\x12B\n" +
	"\bstatuses\x18\x02 \x03(\v2&.daemon.DNSQueryLogStats.StatusesEntryR\bstatuses\x12\x16\n" +
	"\x06bucket\x18\x03 \x01(\x03R\x06bucket\x128\n" +
	"\abuckets\x18\x04 \x03(\v2\x1e.daemon.DNSQueryLogStatsBucketR\abuckets\x12=\n" +
	"\n" +
	"topDomains\x18\x05 \x03(\v2\x1d.daemon.DNSQueryLogStatsCountR\n" +
	"topDomains\x12K\n" +
	"\x11topBlockedDomains\x18\x06 \x03(\v2\x1d.daemon.DNSQueryLogStatsCountR\x11topBlockedDomains\x12=\n" +
	"\n" +
	"topClients\x18\a \x03(\v2\x1d.daemon.DNSQueryLogStatsCountR\n" +
	"topClients\x12=\n" +
	"\n" +
	"queryTypes\x18\b \x03(\v2\x1d.daemon.DNSQueryLogStatsCountR\n" +
	"queryTypes\x12>\n" +
	"\tupstreams\x18\t \x03(\v2 .daemon.DNSQueryLogUpstreamStatsR\tupstreams\x127\n" +
	"\bduration\x18\n" +
	" \x01(\v2\x1b.daemon.DNSQueryLogDurationR\bduration\x12G\n" +
	"\x10upstreamDuration\x18\v \x01(\v2\x1b.daemon.DNSQueryLogDurationR\x10upstreamDuration\x1a;\n" +
	"\rStatusesEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\x03R\x05value:\x028\x01\"\xc9\x01\n" +
	"\x16DNSQueryLogStatsBucket\x12\x12\n" +
	"\x04time\x18\x01 \x01(\x03R\x04time\x12\x14\n" +
	"\x05total\x18\x02 \x01(\x03R\x05total\x12H\n" +
	"\bstatuses\x18\x03 \x03(\v2,.daemon.DNSQueryLogStatsBucket.StatusesEntryR\bstatuses\x1a;\n" +
	"\rStatusesEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\x03R\x05value:\x028\x01\"?\n" +
	"\x15DNSQueryLogStatsCount\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05count\x18\x02 \x01(\x03R\x05count\"\x83\x02\n" +
	"\x18DNSQueryLogUpstreamStats\x12\x16\n" +
	"\x06server\x18\x01 \x01(\tR\x06server\x12\x12\n" +
	"\x04type\x18\x02 \x01(\tR\x04type\x12\x1c\n" +
	"\texchanges\x18\x03 \x01(\x03R\texchanges\x12\x16\n" +
	"\x06failed\x18\x04 \x01(\x03R\x06failed\x12\x1a\n" +
	"\brejected\x18\x05 \x01(\x03R\brejected\x127\n" +
	"\bduration\x18\x06 \x01(\v2\x1b.daemon.DNSQueryLogDurationR\bduration\x12\x18\n" +
	"\aaddress\x18\a \x01(\tR\aaddress\x12\x16\n" +
	"\x06detour\x18\b \x01(\tR\x06detour\"A\n" +
	"\x13DNSQueryLogDuration\x12\x18\n" +
	"\aaverage\x18\x01 \x01(\x03R\aaverage\x12\x10\n" +
	"\x03p95\x18\x02 \x01(\x03R\x03p952\x99\a\n" +
	"\x0fExtendedService\x12J\n" +
	"\x0fGetCapabilities\x12\x16.google.protobuf.Empty\x1a\x1d.daemon.ExtensionCapabilities\"\x00\x12Y\n" +
	"\x16SetTailscaleForceRelay\x12%.daemon.SetTailscaleForceRelayRequest\x1a\x16.google.protobuf.Empty\"\x00\x12c\n" +
	"\"SubscribeTailscaleForceRelayStatus\x12\x16.google.protobuf.Empty\x1a!.daemon.TailscaleForceRelayStatus\"\x000\x01\x12^\n" +
	"\x1aSubscribeOutboundProviders\x12\x16.google.protobuf.Empty\x1a$.daemon.OutboundProviderStatusUpdate\"\x000\x01\x12S\n" +
	"\x16UpdateOutboundProvider\x12\x1f.daemon.OutboundProviderRequest\x1a\x16.google.protobuf.Empty\"\x00\x12K\n" +
	"\x14GetDNSQueryLogStatus\x12\x16.google.protobuf.Empty\x1a\x19.daemon.DNSQueryLogStatus\"\x00\x12I\n" +
	"\x10QueryDNSQueryLog\x12\x1a.daemon.DNSQueryLogRequest\x1a\x17.daemon.DNSQueryLogPage\"\x00\x12R\n" +
	"\x13GetDNSQueryLogStats\x12\x1f.daemon.DNSQueryLogStatsRequest\x1a\x18.daemon.DNSQueryLogStats\"\x00\x12P\n" +
	"\x14SubscribeDNSQueryLog\x12\x1a.daemon.DNSQueryLogRequest\x1a\x18.daemon.DNSQueryLogEntry\"\x000\x01\x12D\n" +
	"\x10ClearDNSQueryLog\x12\x16.google.protobuf.Empty\x1a\x16.google.protobuf.Empty\"\x00\x12A\n" +
	"\rClearDNSCache\x12\x16.google.protobuf.Empty\x1a\x16.google.protobuf.Empty\"\x00B%Z#github.com/sagernet/sing-box/daemonb\x06proto3"

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
	file_daemon_extended_service_proto_msgTypes = make([]protoimpl.MessageInfo, 23)
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
		(*DNSQueryLogStatus)(nil),                // 9: daemon.DNSQueryLogStatus
		(*DNSQueryLogRequest)(nil),               // 10: daemon.DNSQueryLogRequest
		(*DNSQueryLogPage)(nil),                  // 11: daemon.DNSQueryLogPage
		(*DNSQueryLogEntry)(nil),                 // 12: daemon.DNSQueryLogEntry
		(*DNSQueryLogRule)(nil),                  // 13: daemon.DNSQueryLogRule
		(*DNSQueryLogUpstream)(nil),              // 14: daemon.DNSQueryLogUpstream
		(*DNSQueryLogStatsRequest)(nil),          // 15: daemon.DNSQueryLogStatsRequest
		(*DNSQueryLogStats)(nil),                 // 16: daemon.DNSQueryLogStats
		(*DNSQueryLogStatsBucket)(nil),           // 17: daemon.DNSQueryLogStatsBucket
		(*DNSQueryLogStatsCount)(nil),            // 18: daemon.DNSQueryLogStatsCount
		(*DNSQueryLogUpstreamStats)(nil),         // 19: daemon.DNSQueryLogUpstreamStats
		(*DNSQueryLogDuration)(nil),              // 20: daemon.DNSQueryLogDuration
		nil,                                      // 21: daemon.DNSQueryLogStats.StatusesEntry
		nil,                                      // 22: daemon.DNSQueryLogStatsBucket.StatusesEntry
		(*emptypb.Empty)(nil),                    // 23: google.protobuf.Empty
	}
)

var file_daemon_extended_service_proto_depIdxs = []int32{
	1,  // 0: daemon.ExtensionCapabilities.capabilities:type_name -> daemon.ExtensionCapability
	6,  // 1: daemon.OutboundProviderStatusUpdate.providers:type_name -> daemon.OutboundProviderStatus
	7,  // 2: daemon.OutboundProviderStatus.subscriptionInfo:type_name -> daemon.OutboundProviderSubscriptionInfo
	8,  // 3: daemon.OutboundProviderStatus.items:type_name -> daemon.OutboundProviderItem
	12, // 4: daemon.DNSQueryLogPage.entries:type_name -> daemon.DNSQueryLogEntry
	13, // 5: daemon.DNSQueryLogEntry.rules:type_name -> daemon.DNSQueryLogRule
	14, // 6: daemon.DNSQueryLogEntry.upstreams:type_name -> daemon.DNSQueryLogUpstream
	10, // 7: daemon.DNSQueryLogStatsRequest.filter:type_name -> daemon.DNSQueryLogRequest
	21, // 8: daemon.DNSQueryLogStats.statuses:type_name -> daemon.DNSQueryLogStats.StatusesEntry
	17, // 9: daemon.DNSQueryLogStats.buckets:type_name -> daemon.DNSQueryLogStatsBucket
	18, // 10: daemon.DNSQueryLogStats.topDomains:type_name -> daemon.DNSQueryLogStatsCount
	18, // 11: daemon.DNSQueryLogStats.topBlockedDomains:type_name -> daemon.DNSQueryLogStatsCount
	18, // 12: daemon.DNSQueryLogStats.topClients:type_name -> daemon.DNSQueryLogStatsCount
	18, // 13: daemon.DNSQueryLogStats.queryTypes:type_name -> daemon.DNSQueryLogStatsCount
	19, // 14: daemon.DNSQueryLogStats.upstreams:type_name -> daemon.DNSQueryLogUpstreamStats
	20, // 15: daemon.DNSQueryLogStats.duration:type_name -> daemon.DNSQueryLogDuration
	20, // 16: daemon.DNSQueryLogStats.upstreamDuration:type_name -> daemon.DNSQueryLogDuration
	22, // 17: daemon.DNSQueryLogStatsBucket.statuses:type_name -> daemon.DNSQueryLogStatsBucket.StatusesEntry
	20, // 18: daemon.DNSQueryLogUpstreamStats.duration:type_name -> daemon.DNSQueryLogDuration
	23, // 19: daemon.ExtendedService.GetCapabilities:input_type -> google.protobuf.Empty
	2,  // 20: daemon.ExtendedService.SetTailscaleForceRelay:input_type -> daemon.SetTailscaleForceRelayRequest
	23, // 21: daemon.ExtendedService.SubscribeTailscaleForceRelayStatus:input_type -> google.protobuf.Empty
	23, // 22: daemon.ExtendedService.SubscribeOutboundProviders:input_type -> google.protobuf.Empty
	4,  // 23: daemon.ExtendedService.UpdateOutboundProvider:input_type -> daemon.OutboundProviderRequest
	23, // 24: daemon.ExtendedService.GetDNSQueryLogStatus:input_type -> google.protobuf.Empty
	10, // 25: daemon.ExtendedService.QueryDNSQueryLog:input_type -> daemon.DNSQueryLogRequest
	15, // 26: daemon.ExtendedService.GetDNSQueryLogStats:input_type -> daemon.DNSQueryLogStatsRequest
	10, // 27: daemon.ExtendedService.SubscribeDNSQueryLog:input_type -> daemon.DNSQueryLogRequest
	23, // 28: daemon.ExtendedService.ClearDNSQueryLog:input_type -> google.protobuf.Empty
	23, // 29: daemon.ExtendedService.ClearDNSCache:input_type -> google.protobuf.Empty
	0,  // 30: daemon.ExtendedService.GetCapabilities:output_type -> daemon.ExtensionCapabilities
	23, // 31: daemon.ExtendedService.SetTailscaleForceRelay:output_type -> google.protobuf.Empty
	3,  // 32: daemon.ExtendedService.SubscribeTailscaleForceRelayStatus:output_type -> daemon.TailscaleForceRelayStatus
	5,  // 33: daemon.ExtendedService.SubscribeOutboundProviders:output_type -> daemon.OutboundProviderStatusUpdate
	23, // 34: daemon.ExtendedService.UpdateOutboundProvider:output_type -> google.protobuf.Empty
	9,  // 35: daemon.ExtendedService.GetDNSQueryLogStatus:output_type -> daemon.DNSQueryLogStatus
	11, // 36: daemon.ExtendedService.QueryDNSQueryLog:output_type -> daemon.DNSQueryLogPage
	16, // 37: daemon.ExtendedService.GetDNSQueryLogStats:output_type -> daemon.DNSQueryLogStats
	12, // 38: daemon.ExtendedService.SubscribeDNSQueryLog:output_type -> daemon.DNSQueryLogEntry
	23, // 39: daemon.ExtendedService.ClearDNSQueryLog:output_type -> google.protobuf.Empty
	23, // 40: daemon.ExtendedService.ClearDNSCache:output_type -> google.protobuf.Empty
	30, // [30:41] is the sub-list for method output_type
	19, // [19:30] is the sub-list for method input_type
	19, // [19:19] is the sub-list for extension type_name
	19, // [19:19] is the sub-list for extension extendee
	0,  // [0:19] is the sub-list for field type_name
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
			NumMessages:   23,
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
