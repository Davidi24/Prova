package slipvalidation

import (
	"github.com/go-kit/kit/log"
	"nexusws/pkg/nexushttpclient"
)

type EcrSlipValidation struct {
	Header            *SlipRecordHeader `json:"header"`
	MD5               string            `json:"md5"`
	Checksum          string
	rawMessage        []byte
	rawMessageDataLen int //holds the actual number of data, not the length
	l                 log.Logger
	nexusCl           *nexushttpclient.SlipClient
	errorCode         int
}

type SlipRecordHeader struct {
	MessageIdentifier string `json:"MessageIdentifier"`
	ProtocolVersion   string `json:"ProtocolVersion"`
	EcrSerial         string `json:"EcrSerial"`
	NrMac             string `json:"NrMac"`
	RapZ              string `json:"RapZ"`
	DailySlipNo       string `json:"DailySlipNo"`
	SerialSlip        string `json:"SerialSlip"`
}
