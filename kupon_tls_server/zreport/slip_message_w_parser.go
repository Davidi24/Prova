package zreport

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"io"
	"nexusws/pkg/checksum"
	context2 "nexusws/pkg/context"
	"nexusws/pkg/nexus_errors"
	"nexusws/pkg/nexushttpclient"
	"nexusws/pkg/nexushttpclient/zreport"
)

func New(l log.Logger, nexusCl *nexushttpclient.SlipClient, msg []byte, msgLen int) *RawZReport {
	e := &RawZReport{
		rawMessage: make([]byte, ZReportMaxMessageLength),
		l:          l,
		nexusCl:    nexusCl,
	}
	copy(e.rawMessage, msg)
	e.rawMessageDataLen = msgLen

	return e
}
func (s *RawZReport) RawMessage() []byte {
	return s.rawMessage[:s.rawMessageDataLen]
}
func (s *RawZReport) parseMessage() error {

	bodyLen := binary.LittleEndian.Uint16(s.rawMessage[ZReportLengthFieldOffset:ZReportLengthFieldLast])

	//level.Info(s.l).Log("length field value", fmt.Sprintf("%d", bodyLen))

	bodyLen = bodyLen -
		ZReportLengthFieldLength -
		ZReportTypeLength -
		ZReportEcrSerialLength -
		ZReportEcrFileNameLength -
		ZReportCheckSumLength

	//level.Info(s.l).Log("zreport data length", fmt.Sprintf("%d", bodyLen))

	if bodyLen > ZReportMaxMessageLength {
		err := errors.New("total message length is bigger then maximum allowed")
		return err
	}
	s.bodyLength = int(bodyLen)

	fileContent := s.rawMessage[ZReportEcrFileContentOffset : ZReportEcrFileContentOffset+bodyLen]
	enc := base64.StdEncoding.EncodeToString(fileContent)

	isNull := func(c rune) bool {
		return c == 0
	}

	fileNameRaw := make([]byte, ZReportEcrFileNameLength)
	copy(fileNameRaw, s.rawMessage[ZReportEcrFileNameOffset:ZReportEcrFileNameLast])
	r := bytes.TrimFunc(fileNameRaw, isNull)

	s.Report = &zreport.ZReport{
		ProtocolVersion:   string(s.rawMessage[ZReportProtocolOffset:ZReportProtocolLast]),
		ECRSerial:         string(s.rawMessage[ZReportEcrSerialOffset:ZReportEcrSerialLast]),
		FileName:          string(r),
		FileContentBase64: enc,
	}
	return nil
}

func (s *RawZReport) Handle(ctx context.Context, r io.ReadWriter) error {
	logger := log.With(s.l, "zreport", "Handle", "trace_id", context2.GetTraceId(ctx))

	if s.rawMessageDataLen < ZReportMaxMessageLength {
		for {
			if s.rawMessageDataLen >= ZReportHeaderLength {
				err := s.parseMessage()
				if err != nil {
					return err
				}
			}
			//level.Info(logger).Log("raw message length", fmt.Sprintf("%d", s.rawMessageDataLen))
			//level.Info(logger).Log("ZReportHeaderLength + s.bodyLength + ZReportCheckSumLength", fmt.Sprintf("%d", ZReportHeaderLength+s.bodyLength+ZReportCheckSumLength))
			if s.rawMessageDataLen >= (ZReportHeaderLength + s.bodyLength + ZReportCheckSumLength) {
				break
			}
			//level.Info(logger).Log("reading more data", "reading more data")

			headerLen, rerr := r.Read(s.rawMessage[s.rawMessageDataLen:])

			//level.Info(logger).Log("ReadMore", fmt.Sprintf("%d", s.rawMessage[:s.rawMessageDataLen]))

			if rerr != nil {
				if rerr == io.EOF {
					s.rawMessageDataLen += headerLen
					break
				}
				s.errorCode = nexus_errors.ErrUnableToReadDataFromNetwork
				s.sendNack(ctx, r, s.errorCode)
				return rerr
			}

			s.rawMessageDataLen += headerLen

			if s.rawMessageDataLen == ZReportMaxMessageLength {
				break
			} else if uint(s.rawMessageDataLen) > ZReportMaxMessageLength {
				s.errorCode = nexus_errors.ErrReceivedMoreDataThenExpected
				s.sendNack(ctx, r, s.errorCode)

				err := errors.New("received more data then expected from network")
				return err
			}
		}
	}

	err := s.parseMessage()
	if err != nil {
		return err
	}

	//var cs string
	cOffset := ZReportHeaderLength + s.bodyLength

	s.Checksum = string(s.rawMessage[cOffset:s.rawMessageDataLen])

	cs := checksum.CalcXorChecksum(s.rawMessage[ZReportIdentifierOffset : ZReportHeaderLength+s.bodyLength])

	//body, _ := json.Marshal(s.Report)
	//level.Info(logger).Log("info", string(body))

	if cs != s.Checksum {
		err = errors.New("checksums do not match")
		s.sendNack(ctx, r, nexus_errors.ErrChecksumError)
		return err
	}

	req := nexushttpclient.ZReportReq{}
	req.ECRSerial = s.Report.ECRSerial
	req.FileName = s.Report.FileName
	req.FileContentBase64 = s.Report.FileContentBase64

	level.Info(logger).Log("filename", s.Report.FileName)
	//_, err = s.nexusCl.ZReportInsert(ctx, &req)
	////TODO hiq komentin me poshte dhe implement endpoont zReportInsert
	//if err != nil {
	//	//s.sendNack(r, nexus_errors.ErrErrorSavingZReport)
	//	level.Error(logger).Log("err", err)
	//	//return err
	//}
	//if resp.ErrorCode != 0 {
	//	s.sendNack(r, nexus_errors.ErrErrorSavingZReport)
	//	return fmt.Errorf("%v", resp)
	//}

	err = s.sendAck(ctx, r)
	if err != nil {
		return err
	}
	return nil
}

func (s *RawZReport) sendNack(ctx context.Context, r io.ReadWriter, errorCode int) {
	logger := log.With(s.l, "zreport", "sendNack", "trace_id", context2.GetTraceId(ctx))
	ackMsg := fmt.Sprintf("A%04d", errorCode)
	resp := []byte(ackMsg)
	_, err := r.Write(resp) //TODO kthe errorin e duhur
	if err != nil {
		level.Error(logger).Log("err", err)
		return
	}
	//level.Info(logger).Log("sent rawdata: ", hex.EncodeToString(resp))
}
func (s *RawZReport) sendAck(ctx context.Context, r io.ReadWriter) error {
	logger := log.With(s.l, "zreport", "sendAck", "trace_id", context2.GetTraceId(ctx))

	resp := []byte(ZReportProtocolACK)
	_, err := r.Write(resp)
	if err != nil {
		level.Error(logger).Log("err", err)
		return err
	}
	//level.Info(logger).Log("sent rawdata: ", hex.EncodeToString(resp))
	return nil
}
