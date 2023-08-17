package zreport

import (
	"github.com/go-kit/kit/log"
	"nexusws/pkg/nexushttpclient"
	"nexusws/pkg/nexushttpclient/zreport"
)

type RawZReport struct {
	Report   *zreport.ZReport
	Checksum string

	rawMessage        []byte
	rawMessageDataLen int //holds the actual number of data, not the message length
	bodyLength        int
	l                 log.Logger
	nexusCl           *nexushttpclient.SlipClient
	errorCode         int
}
