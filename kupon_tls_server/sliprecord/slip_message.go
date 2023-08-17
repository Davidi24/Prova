package sliprecord

import (
	"github.com/go-kit/kit/log"
	"nexusws/pkg/nexushttpclient"
	v13 "nexusws/pkg/nexushttpclient/v13"
)

type RawEcrSlipRecord struct {
	Header13      *v13.MessageGHeader
	v13SlipRecord *v13.MessageG
	Checksum      string

	rawMessage        []byte
	rawMessageDataLen int //holds the actual number of data, not the message length
	l                 log.Logger
	nexusCl           *nexushttpclient.SlipClient
	errorCode         int
	protVersion       string
}
