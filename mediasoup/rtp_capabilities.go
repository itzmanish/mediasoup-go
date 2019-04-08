package mediasoup

import (
	"github.com/jinzhu/copier"
	"github.com/jiyeyuran/mediasoup-go/mediasoup/h264profile"
)

type RtpCapabilities struct {
	Codecs           []RtpCodecCapability `json:"codecs,omitempty"`
	HeaderExtensions []RtpHeaderExtension `json:"headerExtensions,omitempty"`
	FecMechanisms    []string             `json:"fecMechanisms,omitempty"`
}

type RtpRemoteCapabilities struct {
	Codecs           []RtpMappingCodec     `json:"codecs,omitempty"`
	HeaderExtensions []RtpMappingHeaderExt `json:"headerExtensions,omitempty"`
	Encodings        []RtpMappingEncoding  `json:"encodings,omitempty"`
	Rtcp             RtcpConfiguation      `json:"rtcp,omitempty"`
}

type RtpMappingCodec struct {
	*RtpCodecCapability
	PayloadType       int `json:"payloadType,omitempty"`
	MappedPayloadType int `json:"mappedPayloadType,omitempty"`
}

type RtpMappingHeaderExt struct {
	*RtpHeaderExtension
	Id       int `json:"id,omitempty"`
	MappedId int `json:"mappedId,omitempty"`
}

type RtcpConfiguation struct {
	Cname       string `json:"cname,omitempty"`
	ReducedSize bool   `json:"reducedSize,omitempty"`
	Mux         bool   `json:"mux,omitempty"`
}

type RtpCodecCapability struct {
	Kind                 string         `json:"kind,omitempty"`
	MimeType             string         `json:"mimeType,omitempty"`
	ClockRate            int            `json:"clockRate,omitempty"`
	Channels             int            `json:"channels,omitempty"`
	PreferredPayloadType int            `json:"preferredPayloadType,omitempty"`
	Parameters           *RtpParameter  `json:"parameters,omitempty"`
	RtcpFeedback         []RtcpFeedback `json:"rtcpFeedback,omitempty"`
}

type RtcpFeedback struct {
	Type      string `json:"type,omitempty"`
	Parameter string `json:"parameter,omitempty"`
}

type RtpParameter struct {
	h264profile.RtpH264Parameter
	Apt          int `json:"apt,omitempty"`          // used by rtx codec
	Useinbandfec int `json:"useinbandfec,omitempty"` // used by audio
}

type RtpMappingEncoding struct {
	Rid        uint32              `json:"rid,omitempty"`
	Ssrc       uint32              `json:"ssrc,omitempty"`
	MappedSsrc uint32              `json:"mappedSsrc,omitempty"`
	Rtx        *RtpMappingEncoding `json:"rtx,omitempty"`
}

type RtpHeaderExtension struct {
	Kind             string `json:"kind,omitempty"`
	Uri              string `json:"uri,omitempty"`
	PreferredId      int    `json:"preferredId,omitempty"`      // used by router
	PreferredEncrypt bool   `json:"preferredEncrypt,omitempty"` // used by router
}

var supportedRtpCapabilities = RtpCapabilities{
	Codecs: []RtpCodecCapability{
		{
			Kind:      "audio",
			MimeType:  "audio/opus",
			ClockRate: 48000,
			Channels:  2,
		},
		{
			Kind:                 "audio",
			MimeType:             "audio/PCMU",
			PreferredPayloadType: 0,
			ClockRate:            8000,
		},
		{
			Kind:                 "audio",
			MimeType:             "audio/PCMA",
			PreferredPayloadType: 8,
			ClockRate:            8000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/ISAC",
			ClockRate: 32000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/ISAC",
			ClockRate: 16000,
		},
		{
			Kind:                 "audio",
			MimeType:             "audio/G722",
			PreferredPayloadType: 9,
			ClockRate:            8000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/iLBC",
			ClockRate: 8000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/SILK",
			ClockRate: 24000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/SILK",
			ClockRate: 16000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/SILK",
			ClockRate: 12000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/SILK",
			ClockRate: 8000,
		},
		{
			Kind:                 "audio",
			MimeType:             "audio/CN",
			PreferredPayloadType: 13,
			ClockRate:            32000,
		},
		{
			Kind:                 "audio",
			MimeType:             "audio/CN",
			PreferredPayloadType: 13,
			ClockRate:            16000,
		},
		{
			Kind:                 "audio",
			MimeType:             "audio/CN",
			PreferredPayloadType: 13,
			ClockRate:            8000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/telephone-event",
			ClockRate: 48000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/telephone-event",
			ClockRate: 32000,
		},

		{
			Kind:      "audio",
			MimeType:  "audio/telephone-event",
			ClockRate: 16000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/telephone-event",
			ClockRate: 8000,
		},
		{
			Kind:      "video",
			MimeType:  "video/VP8",
			ClockRate: 90000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
			},
		},
		{
			Kind:      "video",
			MimeType:  "video/VP9",
			ClockRate: 90000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
			},
		},
		{
			Kind:      "video",
			MimeType:  "video/H264",
			ClockRate: 90000,
			Parameters: &RtpParameter{
				RtpH264Parameter: h264profile.RtpH264Parameter{
					PacketizationMode:     1,
					LevelAsymmetryAllowed: 1,
				},
			},
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
			},
		},
		{
			Kind:      "video",
			MimeType:  "video/H264",
			ClockRate: 90000,
			Parameters: &RtpParameter{
				RtpH264Parameter: h264profile.RtpH264Parameter{
					PacketizationMode:     0,
					LevelAsymmetryAllowed: 1,
				},
			},
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
			},
		},
		{
			Kind:      "video",
			MimeType:  "video/H265",
			ClockRate: 90000,
			Parameters: &RtpParameter{
				RtpH264Parameter: h264profile.RtpH264Parameter{
					PacketizationMode:     1,
					LevelAsymmetryAllowed: 1,
				},
			},
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
			},
		},
		{
			Kind:      "video",
			MimeType:  "video/H265",
			ClockRate: 90000,
			Parameters: &RtpParameter{
				RtpH264Parameter: h264profile.RtpH264Parameter{
					PacketizationMode:     0,
					LevelAsymmetryAllowed: 1,
				},
			},
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
			},
		},
	},
	HeaderExtensions: []RtpHeaderExtension{
		{
			Kind:             "audio",
			Uri:              "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
			PreferredId:      1,
			PreferredEncrypt: false,
		},
		{
			Kind:             "video",
			Uri:              "urn:ietf:params:rtp-hdrext:toffset",
			PreferredId:      2,
			PreferredEncrypt: false,
		},
		{
			Kind:             "audio",
			Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
			PreferredId:      3,
			PreferredEncrypt: false,
		},
		{
			Kind:             "video",
			Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
			PreferredId:      3,
			PreferredEncrypt: false,
		},
		{
			Kind:             "video",
			Uri:              "urn:3gpp:video-orientation",
			PreferredId:      4,
			PreferredEncrypt: false,
		},
		{
			Kind:             "audio",
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:mid",
			PreferredId:      5,
			PreferredEncrypt: false,
		},
		{
			Kind:             "video",
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:mid",
			PreferredId:      5,
			PreferredEncrypt: false,
		},
		{
			Kind:             "video",
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id",
			PreferredId:      6,
			PreferredEncrypt: false,
		},
		{
			Kind:             "video",
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id",
			PreferredId:      7,
			PreferredEncrypt: false,
		},
	},
}

func GetSupportedRtpCapabilities() (rtpCapabilities RtpCapabilities) {
	copier.Copy(&rtpCapabilities, &supportedRtpCapabilities)

	return
}
