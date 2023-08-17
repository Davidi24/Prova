package slipvalidation

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"io"
	"nexusws/cmd/kupon_tls_server/sliprecord"
	"nexusws/pkg/checksum"
	"nexusws/pkg/nexus_errors"
	"nexusws/pkg/nexushttpclient"
	v13 "nexusws/pkg/nexushttpclient/v13"
	"strings"
)

func New(l log.Logger, nexusCl *nexushttpclient.SlipClient, msg []byte, msgLen int) *EcrSlipValidation {
	e := &EcrSlipValidation{
		rawMessage: make([]byte, sliprecord.SlipMaxMessageLength),
		l:          l,
		nexusCl:    nexusCl,
	}
	copy(e.rawMessage, msg)
	e.rawMessageDataLen = msgLen

	return e
}
func (s *EcrSlipValidation) RawMessage() ([]byte, int) {
	return s.rawMessage, s.rawMessageDataLen
}
func (s *EcrSlipValidation) Handle(ctx context.Context, r io.ReadWriter) error {

	//if we haven't read the message, try to read it now
	if s.rawMessageDataLen < SlipValidationMaxMessageLength {
		for {
			msgLen, rerr := r.Read(s.rawMessage[s.rawMessageDataLen:])
			if rerr != nil {
				if rerr == io.EOF {
					s.rawMessageDataLen += msgLen
					break
				}
				s.errorCode = nexus_errors.ErrUnableToReadDataFromNetwork
				s.sendNack(r, s.errorCode)
				return rerr
			}

			s.rawMessageDataLen += msgLen
			if s.rawMessageDataLen == SlipValidationMaxMessageLength {
				break
			} else if uint(s.rawMessageDataLen) > SlipValidationMaxMessageLength {
				s.errorCode = nexus_errors.ErrReceivedMoreDataThenExpected
				s.sendNack(r, s.errorCode)

				err := errors.New("received more data then expected from network")
				return err
			}
		}
	}

	err := s.parseMessageEV1()
	if err != nil {
		s.sendNack(r, s.errorCode)
		return err
	}

	s.MD5 = string(s.rawMessage[SlipValidationMD5Offset:SlipValidationMD5Last])

	//todo check md5

	s.Checksum = string(s.rawMessage[SlipValidationCheckSumOffset:SlipValidationCheckSumLast])
	cs := checksum.CalcXorChecksum(s.rawMessage[SlipValidationIdentifierOffset:SlipValidationMD5Last])
	if cs != s.Checksum {
		err = errors.New("checksums do not match")
		s.errorCode = nexus_errors.ErrChecksumError
		s.sendNack(r, s.errorCode)
		return err
	}

	res, err := s.nexusCl.SlipValidationInsert(ctx, &v13.SlipValidationInsertReq{
		Identificationnumber: s.Header.EcrSerial,
		Nrmac:                s.Header.NrMac,
		Nrzreport:            s.Header.RapZ,
		Dailyslipno:          s.Header.DailySlipNo,
		Slipserial:           s.Header.SerialSlip,
		Md5:                  s.MD5,
	})
	if err != nil {
		s.sendNack(r, s.errorCode)
		return err
	}

	if res.ErrorCode != 0 {
		s.sendNack(r, res.ErrorCode)
		return fmt.Errorf("%v", res)
	}

	err = s.sendAck(r)
	if err != nil {
		return err
	}

	//slipRes := NewResponse(res.Data)
	slipRes := NewResponse(s.Header.ProtocolVersion, res.QrCode)
	resp, err := slipRes.MarshalBinary()
	if err != nil {
		s.sendNack(r, s.errorCode)
		return err
	}
	_, err = r.Write(resp)
	if err != nil {
		return err
	}
	return nil
}

func (s *EcrSlipValidation) parseMessageEV1() error {
	if s.rawMessageDataLen < SlipValidationMaxMessageLength {
		return errors.New("message E data less then expected")
	}

	s.Header = &SlipRecordHeader{
		MessageIdentifier: string(s.rawMessage[SlipValidationIdentifierOffset:SlipValidationIdentifierLast]),
		ProtocolVersion:   string(s.rawMessage[SlipValidationProtocolOffset:SlipValidationProtocolLast]),
		EcrSerial:         string(s.rawMessage[SlipValidationEcrSerialdOffset:SlipValidationEcrSerialLast]),
		NrMac:             strings.TrimLeft(string(s.rawMessage[SlipValidationMacOffset:SlipValidationMacLast]), "0"),
		RapZ:              strings.TrimLeft(string(s.rawMessage[SlipValidationRapZOffset:SlipValidationRapZLast]), "0"),
		DailySlipNo:       strings.TrimLeft(string(s.rawMessage[SlipValidationDailySlipNoOffset:SlipValidationDailySlipNoLast]), "0"),
		SerialSlip:        strings.TrimLeft(string(s.rawMessage[SlipValidationSerialOffset:SlipValidationSerialLast]), "0"),
	}
	return nil
}
func (s *EcrSlipValidation) sendNack(r io.ReadWriter, errorCode int) {
	ackMsg := fmt.Sprintf("A%04d", errorCode)
	resp := []byte(ackMsg)
	_, err := r.Write(resp)
	if err != nil {
		level.Error(s.l).Log("err", err)
		return
	}
	//level.Info(s.l).Log("sent rawdata: ", hex.EncodeToString(resp))
}

func (s *EcrSlipValidation) sendAck(r io.ReadWriter) error {
	resp := []byte(sliprecord.SlipRecordProtocolACK)
	_, err := r.Write(resp)
	if err != nil {
		level.Error(s.l).Log("err", err)
		return err
	}
	//level.Info(s.l).Log("sent rawdata: ", hex.EncodeToString(resp))
	return nil
}
