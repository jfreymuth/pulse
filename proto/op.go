package proto

const (
	OpError   = 0
	OpTimeout = 1
	OpReply   = 2

	OpCreatePlaybackStream = 3
	OpDeletePlaybackStream = 4
	OpCreateRecordStream   = 5
	OpDeleteRecordStream   = 6

	OpExit          = 7
	OpAuth          = 8
	OpSetClientName = 9

	OpLookupSink          = 10
	OpLookupSource        = 11
	OpDrainPlaybackStream = 12
	OpStat                = 13
	OpGetPlaybackLatency  = 14
	OpCreateUploadStream  = 15
	OpDeleteUploadStream  = 16
	OpFinishUploadStream  = 17
	OpPlaySample          = 18
	OpRemoveSample        = 19

	OpGetServerInfo           = 20
	OpGetSinkInfo             = 21
	OpGetSinkInfoList         = 22
	OpGetSourceInfo           = 23
	OpGetSourceInfoList       = 24
	OpGetModuleInfo           = 25
	OpGetModuleInfoList       = 26
	OpGetClientInfo           = 27
	OpGetClientInfoList       = 28
	OpGetSinkInputInfo        = 29
	OpGetSinkInputInfoList    = 30
	OpGetSourceOutputInfo     = 31
	OpGetSourceOutputInfoList = 32
	OpGetSampleInfo           = 33
	OpGetSampleInfoList       = 34
	OpSubscribe               = 35

	OpSetSinkVolume         = 36
	OpSetSinkInputVolume    = 37
	OpSetSourceVolume       = 38
	OpSetSinkMute           = 39
	OpSetSourceMute         = 40
	OpCorkPlaybackStream    = 41
	OpFlushPlaybackStream   = 42
	OpTriggerPlaybackStream = 43

	OpSetDefaultSink        = 44
	OpSetDefaultSource      = 45
	OpSetPlaybackStreamName = 46
	OpSetRecordStreamName   = 47
	OpKillClient            = 48
	OpKillSinkInput         = 49
	OpKillSourceOutput      = 50

	OpLoadModule   = 51
	OpUnloadModule = 52

	// 4 obsolete commands

	OpGetRecordLatency     = 57
	OpCorkRecordStream     = 58
	OpFlushRecordStream    = 59
	OpPrebufPlaybackStream = 60

	OpRequest              = 61 // server -> client
	OpOverflow             = 62 // server -> client
	OpUnderflow            = 63 // server -> client
	OpPlaybackStreamKilled = 64 // server -> client
	OpRecordStreamKilled   = 65 // server -> client
	OpSubscribeEvent       = 66 // server -> client

	OpMoveSinkInput                  = 67
	OpMoveSourceOutput               = 68
	OpSetSinkInputMute               = 69
	OpSuspendSink                    = 70
	OpSuspendSource                  = 71
	OpSetPlaybackStreamBufferAttr    = 72
	OpSetRecordStreamBufferAttr      = 73
	OpUpdatePlaybackStreamSampleRate = 74
	OpUpdateRecordStreamSampleRate   = 75

	OpPlaybackStreamSuspended = 76 // server -> client
	OpRecordStreamSuspended   = 77 // server -> client
	OpPlaybackStreamMoved     = 78 // server -> client
	OpRecordStreamMoved       = 79 // server -> client

	OpUpdateRecordStreamProplist   = 80
	OpUpdatePlaybackStreamProplist = 81
	OpUpdateClientProplist         = 82
	OpRemoveRecordStreamProplist   = 83
	OpRemovePlaybackStreamProplist = 84
	OpRemoveClientProplist         = 85

	OpStarted = 86 // server -> client

	OpExtension = 87

	OpGetCardInfo     = 88
	OpGetCardInfoList = 89
	OpSetCardProfile  = 90

	OpClientEvent               = 91 // server -> client
	OpPlaybackStreamEvent       = 92 // server -> client
	OpRecordStreamEvent         = 93 // server -> client
	OpPlaybackBufferAttrChanged = 94 // server -> client
	OpRecordBufferAttrChanged   = 95 // server -> client

	OpSetSinkPort           = 96
	OpSetSourcePort         = 97
	OpSetSourceOutputVolume = 98
	OpSetSourceOutputMute   = 99

	OpSetPortLatencyOffset = 100

	OpEnableSRBChannel  = 101
	OpDisableSRBChannel = 102

	OpRegisterMemfdShmid = 103
)

type RequestArgs interface{ command() uint32 }
type Reply interface{ IsReplyTo() uint32 }

type CreatePlaybackStream struct {
	SampleSpec
	ChannelMap ChannelMap
	SinkIndex  uint32
	SinkName   string

	BufferMaxLength       uint32
	Corked                bool
	BufferTargetLength    uint32
	BufferPrebufferLength uint32
	BufferMinimumRequest  uint32

	SyncID uint32

	ChannelVolumes ChannelVolumes

	NoRemap      bool "12"
	NoRemix      bool "12"
	FixFormat    bool "12"
	FixRate      bool "12"
	FixChannels  bool "12"
	NoMove       bool "12"
	VariableRate bool "12"

	Muted         bool     "13"
	AdjustLatency bool     "13"
	Properties    PropList "13"

	VolumeSet     bool "14"
	EarlyRequests bool "14"

	MutedSet               bool "15"
	DontInhibitAutoSuspend bool "15"
	FailOnSuspend          bool "15"

	RelativeVolume bool "17"

	Passthrough bool "18"

	Formats []FormatInfo "21"
}
type CreatePlaybackStreamReply struct {
	StreamIndex    uint32
	SinkInputIndex uint32
	Missing        uint32

	BufferMaxLength       uint32 "9"
	BufferTargetLength    uint32 "9"
	BufferPrebufferLength uint32 "9"
	BufferMinimumRequest  uint32 "9"

	SampleSpec "12"
	ChannelMap []byte "12"

	SinkIndex     uint32 "12"
	SinkName      string "12"
	SinkSuspended bool   "12"

	SinkLatency Microseconds "13"

	FormatInfo "21"
}

type DeletePlaybackStream struct{ StreamIndex uint32 }

type CreateRecordStream struct {
	SampleSpec
	ChannelMap      ChannelMap
	SourceIndex     uint32
	SourceName      string
	BufferMaxLength uint32
	Corked          bool
	BufferFragSize  uint32

	NoRemap      bool "12"
	NoRemix      bool "12"
	FixFormat    bool "12"
	FixRate      bool "12"
	FixChannels  bool "12"
	NoMove       bool "12"
	VariableRate bool "12"

	PeakDetect         bool     "13"
	AdjustLatency      bool     "13"
	Properties         PropList "13"
	DirectOnInputIndex uint32   "13"

	EarlyRequests bool "14"

	DontInhibitAutoSuspend bool "15"
	FailOnSuspend          bool "15"

	Formats        []FormatInfo   "22"
	ChannelVolumes ChannelVolumes "22"
	Muted          bool           "22"
	VolumeSet      bool           "22"
	MutedSet       bool           "22"
	RelativeVolume bool           "22"
	Passthrough    bool           "22"
}
type CreateRecordStreamReply struct {
	StreamIndex       uint32
	SourceOutputIndex uint32

	BufferMaxLength uint32 "9"
	BufferFragSize  uint32 "9"

	SampleSpec      "12"
	ChannelMap      ChannelMap "12"
	SourceIndex     uint32     "12"
	SourceName      string     "12"
	SourceSuspended bool       "12"

	SourceLatency Microseconds "13"

	FormatInfo "22"
}

type DeleteRecordStream struct{ StreamIndex uint32 }

type Exit struct{}

type Auth struct {
	Version Version
	Cookie  []byte
}
type AuthReply struct {
	Version Version
}

type SetClientName struct {
	Props PropList
}
type SetClientNameReply struct {
	ClientIndex uint32
}

type LookupSink struct{ SinkName string }
type LookupSinkReply struct{ SinkIndex uint32 }

type LookupSource struct{ SourceName string }
type LookupSourceReply struct{ SourceIndex uint32 }

type DrainPlaybackStream struct {
	StreamIndex uint32
}

type Stat struct{}
type StatReply struct {
	NumAllocated    uint32
	AllocatedSize   uint32
	NumAccumulated  uint32
	AccumulatedSize uint32
	SampleCacheSize uint32
}

type GetPlaybackLatency struct {
	StreamIndex uint32
	Time        Time
}
type GetPlaybackLatencyReply struct {
	Latency     Microseconds
	Unused      Microseconds // always 0
	Running     bool
	RequestTime Time
	ReplyTime   Time
	WriteIndex  int64
	ReadIndex   int64

	UnderrunFor uint64 "13"
	PlayingFor  uint64 "13"
}

type GetRecordLatency struct {
	StreamIndex uint32
	Time        Time
}
type GetRecordLatencyReply struct {
	MonitorLatency Microseconds
	Latency        Microseconds
	Running        bool
	RequestTime    Time
	ReplyTime      Time
	WriteIndex     int64
	ReadIndex      int64
}

type CreateUploadStream struct {
	Name string
	SampleSpec
	ChannelMap ChannelMap
	Length     uint32

	Properties PropList "13"
}
type CreateUploadStreamReply struct {
	StreamIndex uint32
	Length      uint32
}

type DeleteUploadStream struct{ StreamIndex uint32 }

type FinishUploadStream struct {
	StreamIndex uint32
}

type PlaySample struct {
	SinkIndex uint32
	SinkName  string
	Volume    uint32
	Name      string

	Properties PropList "13"
}

type RemoveSample struct {
	Name string
}

type GetServerInfo struct{}
type GetServerInfoReply struct {
	PackageName    string
	PackageVersion string
	Username       string
	Hostname       string

	DefaultSampleSpec SampleSpec
	DefaultSinkName   string
	DefaultSourceName string

	Cookie uint32

	DefaultChannelMap ChannelMap "15"
}

type GetSinkInfo struct {
	SinkIndex uint32
	SinkName  string
}
type GetSinkInfoReply struct {
	SinkIndex uint32
	SinkName  string
	Device    string
	SampleSpec
	ChannelMap         ChannelMap
	ModuleIndex        uint32
	ChannelVolumes     ChannelVolumes
	Mute               bool
	MonitorSourceIndex uint32
	MonitorSourceName  string
	Latency            Microseconds
	Driver             string
	Flags              uint32

	Properties       PropList     "13"
	RequestedLatency Microseconds "13"

	BaseVolume     Volume "15"
	State          uint32 "15"
	NumVolumeSteps uint32 "15"
	CardIndex      uint32 "15"

	Ports []struct {
		Name        string
		Description string
		Priority    uint32
		Available   uint32 "24"
	} "16"
	ActivePortName string "16"

	Formats []FormatInfo "21"
}

type GetSourceInfo struct {
	SourceIndex uint32
	SourceName  string
}
type GetSourceInfoReply struct {
	SourceIndex uint32
	SourceName  string
	Device      string
	SampleSpec
	ChannelMap         ChannelMap
	ModuleIndex        uint32
	ChannelVolumes     ChannelVolumes
	Mute               bool
	MonitorSourceIndex uint32
	MonitorSourceName  string
	Latency            Microseconds
	Driver             string
	Flags              uint32

	Properties       PropList     "13"
	RequestedLatency Microseconds "13"

	BaseVolume     Volume "15"
	State          uint32 "15"
	NumVolumeSteps uint32 "15"
	CardIndex      uint32 "15"

	Ports []struct {
		Name        string
		Description string
		Priority    uint32
		Available   uint32 "24"
	} "16"
	ActivePortName string "16"

	Formats []FormatInfo "21"
}

type GetClientInfo struct{ ClientIndex uint32 }
type GetClientInfoReply struct {
	ClientIndex uint32
	Application string
	ModuleIndex uint32
	Driver      string

	Properties PropList "13"
}

type GetCardInfo struct{ CardIndex uint32 }
type GetCardInfoReply struct {
	CardIndex   uint32
	CardName    string
	ModuleIndex uint32
	Driver      string

	Profiles []struct {
		Name        string
		Description string
		NumSinks    uint32
		NumSources  uint32
		Priority    uint32
		Available   uint32 "29"
	}
	ActiveProfileName string
	Properties        PropList

	Ports []struct {
		Name        string
		Description string
		Priority    uint32
		Available   uint32
		Direction   byte
		Properties  PropList
		Profiles    []struct {
			Name string
		}
		LatencyOffset int64 "27"
	} "26"
}

type GetModuleInfo struct{ ModuleIndex uint32 }
type GetModuleInfoReply struct {
	ModuleIndex uint32
	ModuleName  string
	ModuleArgs  string
	Users       uint32

	Properties PropList "15"
	AutoLoad   bool     "<15"
}

type GetSinkInputInfo struct{ SinkInputIndex uint32 }
type GetSinkInputInfoReply struct {
	SinkInputIndex uint32
	MediaName      string
	ModuleIndex    uint32
	ClientIndex    uint32
	SinkIndex      uint32
	SampleSpec
	ChannelMap     ChannelMap
	ChannelVolumes ChannelVolumes

	SinkInputLatency Microseconds
	SinkLatency      Microseconds
	ResampleMethod   string
	Driver           string

	Muted bool "11"

	Properties PropList "13"

	Corked bool "19"

	VolumeReadable bool "20"
	VolumeWritable bool "20"

	FormatInfo "21"
}

type GetSourceOutputInfo struct{ SourceOutpuIndex uint32 }
type GetSourceOutputInfoReply struct {
	SourceOutpuIndex uint32
	MediaName        string
	ModuleIndex      uint32
	ClientIndex      uint32
	SourceIndex      uint32
	SampleSpec
	ChannelMap ChannelMap

	SourceOutpuLatency Microseconds
	SourceLatency      Microseconds
	ResampleMethod     string
	Driver             string

	Properties PropList "13"

	Corked bool "19"

	ChannelVolumes ChannelVolumes "22"
	Muted          bool           "22"
	VolumeReadable bool           "22"
	VolumeWritable bool           "22"
	FormatInfo     "22"
}

type GetSampleInfo struct {
	SampleIndex uint32
	SampleName  string
}
type GetSampleInfoReply struct {
	SampleIndex    uint32
	SampleName     string
	ChannelVolumes ChannelVolumes
	Duration       Microseconds
	SampleSpec
	ChannelMap ChannelMap
	Length     uint32
	Lazy       bool
	Filename   string

	Properties PropList "13"
}

type GetSinkInfoList struct{}
type GetSourceInfoList struct{}
type GetModuleInfoList struct{}
type GetClientInfoList struct{}
type GetCardInfoList struct{}
type GetSinkInputInfoList struct{}
type GetSourceOutputInfoList struct{}
type GetSampleInfoList struct{}

type GetSinkInfoListReply []*GetSinkInfoReply
type GetSourceInfoListReply []*GetSourceInfoReply
type GetModuleInfoListReply []*GetModuleInfoReply
type GetClientInfoListReply []*GetClientInfoReply
type GetCardInfoListReply []*GetCardInfoReply
type GetSinkInputInfoListReply []*GetSinkInputInfoReply
type GetSourceOutputInfoListReply []*GetSourceOutputInfoReply
type GetSampleInfoListReply []*GetSampleInfoReply

type Subscribe struct{ Mask uint32 }

type SetSinkVolume struct {
	SinkIndex      uint32
	SinkName       string
	ChannelVolumes ChannelVolumes
}

type SetSourceVolume struct {
	SourceIndex    uint32
	SourceName     string
	ChannelVolumes ChannelVolumes
}

type SetSinkInputVolume struct {
	SinkInputIndex uint32
	ChannelVolumes ChannelVolumes
}

type SetSourceOutputVolume struct {
	SourceOutputIndex uint32
	ChannelVolumes    ChannelVolumes
}

type SetSinkMute struct {
	SinkIndex uint32
	SinkName  string
	Mute      bool
}

type SetSourceMute struct {
	SourceIndex uint32
	SourceName  string
	Mute        bool
}

type SetSinkInputMute struct {
	SinkInputIndex uint32
	Mute           bool
}

type SetSourceOutputMute struct {
	SourceOutputIndex uint32
	Mute              bool
}

type CorkPlaybackStream struct {
	StreamIndex uint32
	Corked      bool
}

type CorkRecordStream struct {
	StreamIndex uint32
	Corked      bool
}

type FlushRecordStream struct{ StreamIndex uint32 }
type TriggerPlaybackStream struct{ StreamIndex uint32 }
type FlushPlaybackStream struct{ StreamIndex uint32 }
type PrebufPlaybackStream struct{ StreamIndex uint32 }

type SetPlaybackStreamBufferAttr struct {
	StreamIndex           uint32
	BufferMaxLength       uint32
	BufferTargetLength    uint32
	BufferPrebufferLength uint32
	BufferMinimumRequest  uint32

	AdjustLatency bool "13"

	EarlyRequests bool "14"
}
type SetPlaybackStreamBufferAttrReply struct {
	BufferMaxLength       uint32
	BufferTargetLength    uint32
	BufferPrebufferLength uint32
	BufferMinimumRequest  uint32

	SinkLatency Microseconds "13"
}

type SetRecordStreamBufferAttr struct {
	StreamIndex     uint32
	BufferMaxLength uint32
	BufferFragSize  uint32

	AdjustLatency bool "13"

	EarlyRequests bool "14"
}
type SetRecordStreamBufferAttrReply struct {
	BufferMaxLength uint32
	BufferFragSize  uint32

	SourceLatency Microseconds "13"
}

type UpdatePlaybackStreamSampleRate struct {
	StreamIndex uint32
	SampleRate  uint32
}

type UpdateRecordStreamSampleRate struct {
	StreamIndex uint32
	SampleRate  uint32
}

type UpdatePlaybackStreamProplist struct {
	StreamIndex uint32
	Mode        uint32
	Properties  PropList
}

type UpdateRecordStreamProplist struct {
	StreamIndex uint32
	Mode        uint32
	Properties  PropList
}

type UpdateClientProplist struct {
	Mode       uint32
	Properties PropList
}

type RemovePlaybackStreamProplist struct {
	StreamIndex uint32
	Properties  PropList // ignored
}
type RemoveRecordStreamProplist struct {
	StreamIndex uint32
	Properties  PropList // ignored
}
type RemoveClientProplist struct {
	Properties PropList // ignored
}

type SetDefaultSink struct{ SinkName string }
type SetDefaultSource struct{ SourceName string }

type SetPlaybackStreamName struct {
	StreamIndex uint32
	Name        string
}

type SetRecordStreamName struct {
	StreamIndex uint32
	Name        string
}

type KillSinkInput struct{ SinkInputIndex uint32 }
type KillSourceOutput struct{ SourceOutputIndex uint32 }
type KillClient struct{ ClientIndex uint32 }

type LoadModule struct {
	Name string
	Args string
}
type LoadModuleReply struct {
	ModuleIndex uint32
}

type UnloadModule struct{ ModuleIndex uint32 }

type MoveSinkInput struct {
	SinkInputIndex uint32
	DeviceIndex    uint32
	DeviceName     string
}

type MoveSourceOutput struct {
	SourceOutputIndex uint32
	DeviceIndex       uint32
	DeviceName        string
}

type SuspendSink struct {
	SinkIndex uint32
	SinkName  string
	Suspend   bool
}
type SuspendSource struct {
	SourceIndex uint32
	SourceName  string
	Suspend     bool
}

// The reply type for this command is extension-specific
type Extension struct {
	Index uint32
	Name  string
}

type SetCardProfile struct {
	CardIndex   uint32
	CardName    string
	ProfileName string
}

type SetSinkPort struct {
	SinkIndex uint32
	SinkName  string
	Port      string
}

type SetSourcePort struct {
	SourceIndex uint32
	SourceName  string
	Port        string
}

type SetPortLatencyOffset struct {
	CardIndex uint32
	CardName  string
	PortName  string
	Offset    int64
}

func (*CreatePlaybackStream) command() uint32           { return OpCreatePlaybackStream }
func (*DeletePlaybackStream) command() uint32           { return OpDeletePlaybackStream }
func (*CreateRecordStream) command() uint32             { return OpCreateRecordStream }
func (*DeleteRecordStream) command() uint32             { return OpDeleteRecordStream }
func (*Exit) command() uint32                           { return OpExit }
func (*Auth) command() uint32                           { return OpAuth }
func (*SetClientName) command() uint32                  { return OpSetClientName }
func (*LookupSink) command() uint32                     { return OpLookupSink }
func (*LookupSource) command() uint32                   { return OpLookupSource }
func (*DrainPlaybackStream) command() uint32            { return OpDrainPlaybackStream }
func (*Stat) command() uint32                           { return OpStat }
func (*GetPlaybackLatency) command() uint32             { return OpGetPlaybackLatency }
func (*CreateUploadStream) command() uint32             { return OpCreateUploadStream }
func (*DeleteUploadStream) command() uint32             { return OpDeleteUploadStream }
func (*FinishUploadStream) command() uint32             { return OpFinishUploadStream }
func (*PlaySample) command() uint32                     { return OpPlaySample }
func (*RemoveSample) command() uint32                   { return OpRemoveSample }
func (*GetServerInfo) command() uint32                  { return OpGetServerInfo }
func (*GetSinkInfo) command() uint32                    { return OpGetSinkInfo }
func (*GetSinkInfoList) command() uint32                { return OpGetSinkInfoList }
func (*GetSourceInfo) command() uint32                  { return OpGetSourceInfo }
func (*GetSourceInfoList) command() uint32              { return OpGetSourceInfoList }
func (*GetModuleInfo) command() uint32                  { return OpGetModuleInfo }
func (*GetModuleInfoList) command() uint32              { return OpGetModuleInfoList }
func (*GetClientInfo) command() uint32                  { return OpGetClientInfo }
func (*GetClientInfoList) command() uint32              { return OpGetClientInfoList }
func (*GetSinkInputInfo) command() uint32               { return OpGetSinkInputInfo }
func (*GetSinkInputInfoList) command() uint32           { return OpGetSinkInputInfoList }
func (*GetSourceOutputInfo) command() uint32            { return OpGetSourceOutputInfo }
func (*GetSourceOutputInfoList) command() uint32        { return OpGetSourceOutputInfoList }
func (*GetSampleInfo) command() uint32                  { return OpGetSampleInfo }
func (*GetSampleInfoList) command() uint32              { return OpGetSampleInfoList }
func (*Subscribe) command() uint32                      { return OpSubscribe }
func (*SetSinkVolume) command() uint32                  { return OpSetSinkVolume }
func (*SetSinkInputVolume) command() uint32             { return OpSetSinkInputVolume }
func (*SetSourceVolume) command() uint32                { return OpSetSourceVolume }
func (*SetSinkMute) command() uint32                    { return OpSetSinkMute }
func (*SetSourceMute) command() uint32                  { return OpSetSourceMute }
func (*CorkPlaybackStream) command() uint32             { return OpCorkPlaybackStream }
func (*FlushPlaybackStream) command() uint32            { return OpFlushPlaybackStream }
func (*TriggerPlaybackStream) command() uint32          { return OpTriggerPlaybackStream }
func (*SetDefaultSink) command() uint32                 { return OpSetDefaultSink }
func (*SetDefaultSource) command() uint32               { return OpSetDefaultSource }
func (*SetPlaybackStreamName) command() uint32          { return OpSetPlaybackStreamName }
func (*SetRecordStreamName) command() uint32            { return OpSetRecordStreamName }
func (*KillClient) command() uint32                     { return OpKillClient }
func (*KillSinkInput) command() uint32                  { return OpKillSinkInput }
func (*KillSourceOutput) command() uint32               { return OpKillSourceOutput }
func (*LoadModule) command() uint32                     { return OpLoadModule }
func (*UnloadModule) command() uint32                   { return OpUnloadModule }
func (*GetRecordLatency) command() uint32               { return OpGetRecordLatency }
func (*CorkRecordStream) command() uint32               { return OpCorkRecordStream }
func (*FlushRecordStream) command() uint32              { return OpFlushRecordStream }
func (*PrebufPlaybackStream) command() uint32           { return OpPrebufPlaybackStream }
func (*MoveSinkInput) command() uint32                  { return OpMoveSinkInput }
func (*MoveSourceOutput) command() uint32               { return OpMoveSourceOutput }
func (*SetSinkInputMute) command() uint32               { return OpSetSinkInputMute }
func (*SuspendSink) command() uint32                    { return OpSuspendSink }
func (*SuspendSource) command() uint32                  { return OpSuspendSource }
func (*SetPlaybackStreamBufferAttr) command() uint32    { return OpSetPlaybackStreamBufferAttr }
func (*SetRecordStreamBufferAttr) command() uint32      { return OpSetRecordStreamBufferAttr }
func (*UpdatePlaybackStreamSampleRate) command() uint32 { return OpUpdatePlaybackStreamSampleRate }
func (*UpdateRecordStreamSampleRate) command() uint32   { return OpUpdateRecordStreamSampleRate }
func (*UpdateRecordStreamProplist) command() uint32     { return OpUpdateRecordStreamProplist }
func (*UpdatePlaybackStreamProplist) command() uint32   { return OpUpdatePlaybackStreamProplist }
func (*UpdateClientProplist) command() uint32           { return OpUpdateClientProplist }
func (*RemoveRecordStreamProplist) command() uint32     { return OpRemoveRecordStreamProplist }
func (*RemovePlaybackStreamProplist) command() uint32   { return OpRemovePlaybackStreamProplist }
func (*RemoveClientProplist) command() uint32           { return OpRemoveClientProplist }
func (*Extension) command() uint32                      { return OpExtension }
func (*GetCardInfo) command() uint32                    { return OpGetCardInfo }
func (*GetCardInfoList) command() uint32                { return OpGetCardInfoList }
func (*SetCardProfile) command() uint32                 { return OpSetCardProfile }
func (*SetSinkPort) command() uint32                    { return OpSetSinkPort }
func (*SetSourcePort) command() uint32                  { return OpSetSourcePort }
func (*SetSourceOutputVolume) command() uint32          { return OpSetSourceOutputVolume }
func (*SetSourceOutputMute) command() uint32            { return OpSetSourceOutputMute }
func (*SetPortLatencyOffset) command() uint32           { return OpSetPortLatencyOffset }

func (*CreatePlaybackStreamReply) IsReplyTo() uint32        { return OpCreatePlaybackStream }
func (*CreateRecordStreamReply) IsReplyTo() uint32          { return OpCreateRecordStream }
func (*AuthReply) IsReplyTo() uint32                        { return OpAuth }
func (*SetClientNameReply) IsReplyTo() uint32               { return OpSetClientName }
func (*LookupSinkReply) IsReplyTo() uint32                  { return OpLookupSink }
func (*LookupSourceReply) IsReplyTo() uint32                { return OpLookupSource }
func (*StatReply) IsReplyTo() uint32                        { return OpStat }
func (*GetPlaybackLatencyReply) IsReplyTo() uint32          { return OpGetPlaybackLatency }
func (*CreateUploadStreamReply) IsReplyTo() uint32          { return OpCreateUploadStream }
func (*GetServerInfoReply) IsReplyTo() uint32               { return OpGetServerInfo }
func (*GetSinkInfoReply) IsReplyTo() uint32                 { return OpGetSinkInfo }
func (*GetSinkInfoListReply) IsReplyTo() uint32             { return OpGetSinkInfoList }
func (*GetSourceInfoReply) IsReplyTo() uint32               { return OpGetSourceInfo }
func (*GetSourceInfoListReply) IsReplyTo() uint32           { return OpGetSourceInfoList }
func (*GetModuleInfoReply) IsReplyTo() uint32               { return OpGetModuleInfo }
func (*GetModuleInfoListReply) IsReplyTo() uint32           { return OpGetModuleInfoList }
func (*GetClientInfoReply) IsReplyTo() uint32               { return OpGetClientInfo }
func (*GetClientInfoListReply) IsReplyTo() uint32           { return OpGetClientInfoList }
func (*GetSinkInputInfoReply) IsReplyTo() uint32            { return OpGetSinkInputInfo }
func (*GetSinkInputInfoListReply) IsReplyTo() uint32        { return OpGetSinkInputInfoList }
func (*GetSourceOutputInfoReply) IsReplyTo() uint32         { return OpGetSourceOutputInfo }
func (*GetSourceOutputInfoListReply) IsReplyTo() uint32     { return OpGetSourceOutputInfoList }
func (*GetSampleInfoReply) IsReplyTo() uint32               { return OpGetSampleInfo }
func (*GetSampleInfoListReply) IsReplyTo() uint32           { return OpGetSampleInfoList }
func (*LoadModuleReply) IsReplyTo() uint32                  { return OpLoadModule }
func (*GetRecordLatencyReply) IsReplyTo() uint32            { return OpGetRecordLatency }
func (*SetPlaybackStreamBufferAttrReply) IsReplyTo() uint32 { return OpSetPlaybackStreamBufferAttr }
func (*SetRecordStreamBufferAttrReply) IsReplyTo() uint32   { return OpSetRecordStreamBufferAttr }
func (*GetCardInfoReply) IsReplyTo() uint32                 { return OpGetCardInfo }
func (*GetCardInfoListReply) IsReplyTo() uint32             { return OpGetCardInfoList }

// SERVER -> CLIENT MESSAGES

type Request struct {
	StreamIndex uint32
	Length      uint32
}

type Overflow struct {
	StreamIndex uint32
}

type Underflow struct {
	StreamIndex uint32
	Offset      int64 "23"
}

type PlaybackStreamKilled struct{ StreamIndex uint32 }
type RecordStreamKilled struct{ StreamIndex uint32 }

type SubscribeEvent struct {
	Event uint32
	Index uint32
}

type PlaybackStreamSuspended struct {
	StreamIndex uint32
	Suspended   bool
}

type RecordStreamSuspended struct {
	StreamIndex uint32
	Suspended   bool
}

type PlaybackStreamMoved struct {
	StreamIndex uint32
	DestIndex   uint32
	DestName    string
	Suspended   bool

	BufferMaxLength       uint32       "13"
	BufferTargetLength    uint32       "13"
	BufferPrebufferLength uint32       "13"
	BufferMinimumRequest  uint32       "13"
	SinkLatency           Microseconds "13"
}

type RecordStreamMoved struct {
	StreamIndex uint32
	DestIndex   uint32
	DestName    string
	Suspended   bool

	BufferMaxLength uint32       "13"
	BufferFragSize  uint32       "13"
	SourceLatency   Microseconds "13"
}

type Started struct{ StreamIndex uint32 }

type ClientEvent struct {
	Event      string
	Properties PropList
}

type PlaybackStreamEvent struct {
	StreamIndex uint32
	Event       string
	Properties  PropList
}

type RecordStreamEvent struct {
	StreamIndex uint32
	Event       string
	Properties  PropList
}

type PlaybackBufferAttrChanged struct {
	StreamIndex           uint32
	BufferMaxLength       uint32
	BufferTargetLength    uint32
	BufferPrebufferLength uint32
	BufferMinimumRequest  uint32
	SinkLatency           Microseconds
}
