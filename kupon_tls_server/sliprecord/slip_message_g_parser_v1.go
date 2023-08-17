package sliprecord

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"io"
	"nexusws/pkg/checksum"
	"nexusws/pkg/nexus_errors"
	"nexusws/pkg/nexushttpclient"
	v13 "nexusws/pkg/nexushttpclient/v13"
)

func New(l log.Logger, nexusCl *nexushttpclient.SlipClient, msg []byte, msgLen int) *RawEcrSlipRecord {
	e := &RawEcrSlipRecord{
		rawMessage: make([]byte, SlipMaxMessageLength),
		l:          l,
		nexusCl:    nexusCl,
	}
	copy(e.rawMessage, msg)
	e.rawMessageDataLen = msgLen

	return e
}
func (s *RawEcrSlipRecord) RawMessage() []byte {
	return s.rawMessage[:s.rawMessageDataLen]
}

func (s *RawEcrSlipRecord) parseMessageGV13Header(headerLength int) error {
	if s.rawMessageDataLen < headerLength {
		return errors.New("header data less then expected")
	}

	bodyLen := binary.LittleEndian.Uint16(s.rawMessage[SlipRecordLengthFieldOffset:SlipRecordLengthFieldLast])
	s.Header13 = &v13.MessageGHeader{
		MessageIdentifier: string(s.rawMessage[SlipRecordIdentifierOffset:SlipRecordIdentifierLast]),
		ProtocolVersion:   string(s.rawMessage[SlipRecordProtocolOffset:SlipRecordProtocolLast]),
		MessageLength:     int(bodyLen),
		TypeIdentifier:    string(s.rawMessage[SlipRecordTypeOffset:SlipRecordTypeLast]),
		EcrSerial:         string(s.rawMessage[SlipRecordEcrSerialOffset:SlipRecordEcrSerialLast]),
	}
	return nil
}

func (s *RawEcrSlipRecord) HandleMsgG(ctx context.Context, r io.ReadWriter) error {

	s.protVersion = string(s.rawMessage[SlipRecordProtocolOffset:SlipRecordProtocolLast])

	headerLength := SlipRecordV13HeaderLength

	//if we haven't read the whole header, try to read it now
	if s.rawMessageDataLen < headerLength {
		for {
			headerLen, rerr := r.Read(s.rawMessage[s.rawMessageDataLen:])
			if rerr != nil {
				if rerr == io.EOF {
					s.rawMessageDataLen += headerLen
					break
				}
				s.errorCode = nexus_errors.ErrUnableToReadDataFromNetwork
				s.sendNack(r, s.errorCode)
				return rerr
			}

			s.rawMessageDataLen += headerLen
			if s.rawMessageDataLen >= headerLength { //meaning we have already read past the header
				break
			}
		}
	}

	err := s.parseMessageGV13Header(headerLength)
	if err != nil {
		return err
	}

	totalMessageLengthExpected := s.Header13.MessageLength + headerLength + SlipRecordCheckSumLength

	if totalMessageLengthExpected > SlipMaxMessageLength {
		err := errors.New("total message length is bigger then maximum allowed")
		return err
	}

	// if we haven't reveived the total message then try here
	if s.rawMessageDataLen < totalMessageLengthExpected {
		var bodyLen int
		var rerr error
		for {
			bodyLen, rerr = r.Read(s.rawMessage[s.rawMessageDataLen:])
			if rerr != nil {
				if rerr == io.EOF {
					s.rawMessageDataLen += bodyLen
					break
				}
				s.errorCode = nexus_errors.ErrUnableToReadDataFromNetwork
				s.sendNack(r, s.errorCode)
				return rerr
			}

			s.rawMessageDataLen += bodyLen
			if s.rawMessageDataLen == totalMessageLengthExpected {
				break
			} else if s.rawMessageDataLen > totalMessageLengthExpected {
				s.errorCode = nexus_errors.ErrReceivedMoreDataThenExpected
				s.sendNack(r, s.errorCode)

				err := errors.New("received more data then expected from network")
				return err
			}
		}
	}
	if s.rawMessageDataLen != totalMessageLengthExpected {
		err := fmt.Errorf("unable to read all data from network, expected %d bytes received %d", totalMessageLengthExpected, s.rawMessageDataLen)
		s.sendNack(r, nexus_errors.ErrUnableToReadDataFromNetwork)
		return err
	}

	err = s.parseBody(ctx, headerLength)
	if err != nil {

		d := s.RawMessage()
		level.Error(s.l).Log("received raw message G:", hex.EncodeToString(d))
		level.Error(s.l).Log("received plain message G:", string(d))

		level.Error(s.l).Log("error", err)
		s.sendNack(r, s.errorCode)
		return err
	}

	cOffset := headerLength + s.Header13.MessageLength

	s.Checksum = string(s.rawMessage[cOffset:s.rawMessageDataLen])

	cs := checksum.CalcXorChecksum(s.rawMessage[SlipRecordIdentifierOffset : headerLength+s.Header13.MessageLength])
	if cs != s.Checksum {
		err = errors.New("checksums do not match")
		s.sendNack(r, nexus_errors.ErrChecksumError)

		level.Error(s.l).Log("error", err)
		return err
	}

	body, _ := json.Marshal(s.v13SlipRecord)
	//level.Info(s.l).Log("info", string(body))

	sr := &nexushttpclient.SlipDataRawInsertReq{
		Ecridentificationnumber: s.v13SlipRecord.Header.EcrSerial,
		Firstrecordnumber:       "",
		Lastrecordnumber:        "",
		Recordtype:              s.v13SlipRecord.Header.TypeIdentifier,
		RawSlipData:             hex.EncodeToString(s.rawMessage[:s.rawMessageDataLen]),
	}
	sresp, err := s.nexusCl.SlipRawInsert(ctx, sr)
	if err != nil {
		return err
	}
	if sresp.ErrorCode != 0 {
		err = errors.New(sresp.ErrorMessage)

		level.Error(s.l).Log("info", string(body))
		level.Error(s.l).Log("error", errors.New(sresp.ErrorMessage))
		s.sendNack(r, nexus_errors.ErrWSSlipError)
		return err
	}

	sresp, err = s.nexusCl.SlipJSONInsert13(ctx, s.v13SlipRecord)

	if err != nil {
		level.Info(s.l).Log("info", string(body))
		level.Error(s.l).Log("error", err)

		s.sendNack(r, nexus_errors.ErrUnableToSaveSlipData)
		return err
	}
	if sresp.ErrorCode != 0 {
		level.Error(s.l).Log("err", string(body))
		level.Error(s.l).Log("err", errors.New(sresp.ErrorMessage))

		s.sendNack(r, nexus_errors.ErrWSSlipError)
		return errors.New(sresp.ErrorMessage)
	}

	err = s.sendAck(r)
	if err != nil {
		return err
	}
	return nil
}

func (s *RawEcrSlipRecord) parseBody(ctx context.Context, headerLength int) error {

	body := s.rawMessage[headerLength : headerLength+s.Header13.MessageLength]

	if s.Header13.TypeIdentifier == "4" { // multi record

		s.v13SlipRecord = &v13.MessageG{
			Header:   s.Header13,
			Records:  make([]v13.SlipRecord, 0),
			Checksum: s.Checksum,
		}
		return s.parseV13(body)

	} else if s.Header13.TypeIdentifier == "5" { // request lottery data
		// TODO
		s.errorCode = nexus_errors.ErrUnimplementedFunction
		return errors.New("unimplemented lottery request")

	} else {
		s.errorCode = nexus_errors.ErrUnknownTypeIdentifier
		return fmt.Errorf("unknown message TypeIdentifier %s", s.Header13.TypeIdentifier)
	}

	return nil
}

func (s *RawEcrSlipRecord) sendNack(r io.ReadWriter, errorCode int) {
	ackMsg := fmt.Sprintf("A%04d", errorCode)
	resp := []byte(ackMsg)
	_, err := r.Write(resp) //TODO kthe errorin e duhur
	if err != nil {
		level.Error(s.l).Log("err", err)
		return
	}
	level.Info(s.l).Log("sent rawdata: ", hex.EncodeToString(resp))
}
func (s *RawEcrSlipRecord) sendAck(r io.ReadWriter) error {
	resp := []byte(SlipRecordProtocolACK)
	_, err := r.Write(resp)
	if err != nil {
		level.Error(s.l).Log("err", err)
		return err
	}
	//level.Info(s.l).Log("sent rawdata: ", hex.EncodeToString(resp))
	return nil
}

func (s *RawEcrSlipRecord) parseV13(data []byte) error {

	splitLinesFunc := func(c rune) bool {
		return c == '\n'
	}
	rawlines := bytes.FieldsFunc(data, splitLinesFunc)

	isNull := func(c rune) bool {
		return c == 0
	}
	for _, r := range rawlines {
		// remove null characters
		r = bytes.TrimFunc(r, isNull)
		if r != nil {
			switch line := r[0]; line {
			case 'A', 'a':
				lineA, h, err := v13.NewLineA(r[1:])
				if err != nil {
					s.errorCode = nexus_errors.ErrWrongNumberOfFields
					return err
				}
				foundLine := false
				for i, t := range s.v13SlipRecord.Records {
					if t.DailySlipNo == h.DailySlipNo &&
						t.SlipSerial == h.SlipSerial &&
						t.ZReport == h.ZReport &&
						t.Mac == h.Mac {
						s.v13SlipRecord.Records[i].LineA = lineA
						foundLine = true
					}
				}
				if !foundLine {
					sr := v13.SlipRecord{
						LineA:                     lineA,
						IsTransmittedInBackground: line == 'a',
					}
					sr.Mac = h.Mac
					sr.ZReport = h.ZReport
					sr.SlipSerial = h.SlipSerial
					sr.DailySlipNo = h.DailySlipNo
					s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
				}
			case 'B', 'b':
				{
					line1, h, err := v13.NewLineB(r[1:])
					if err != nil {
						s.errorCode = nexus_errors.ErrWrongNumberOfFields
						return err
					}
					foundLine := false

					for i, t := range s.v13SlipRecord.Records {
						if t.DailySlipNo == h.DailySlipNo &&
							t.SlipSerial == h.SlipSerial &&
							t.ZReport == h.ZReport &&
							t.Mac == h.Mac {
							s.v13SlipRecord.Records[i].LineB = append(s.v13SlipRecord.Records[i].LineB, *line1)
							foundLine = true
						}
					}

					if !foundLine {
						sr := v13.SlipRecord{
							LineB:                     make([]v13.LineB, 0),
							IsTransmittedInBackground: line == 'b',
						}
						sr.Mac = h.Mac
						sr.ZReport = h.ZReport
						sr.SlipSerial = h.SlipSerial
						sr.DailySlipNo = h.DailySlipNo
						s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
						recordIndex := len(s.v13SlipRecord.Records) - 1
						s.v13SlipRecord.Records[recordIndex].LineB = append(s.v13SlipRecord.Records[recordIndex].LineB, *line1)
					}
				}
			case 'C', 'c':
				{
					line2, h, err := v13.NewLineC(r[1:])
					if err != nil {
						s.errorCode = nexus_errors.ErrWrongNumberOfFields
						return err
					}
					foundLine := false
					for i, t := range s.v13SlipRecord.Records {
						if t.DailySlipNo == h.DailySlipNo &&
							t.SlipSerial == h.SlipSerial &&
							t.ZReport == h.ZReport &&
							t.Mac == h.Mac {
							s.v13SlipRecord.Records[i].LineC = append(s.v13SlipRecord.Records[i].LineC, *line2)

							foundLine = true
						}
					}
					if !foundLine {
						sr := v13.SlipRecord{
							LineC:                     make([]v13.LineC, 0),
							IsTransmittedInBackground: line == 'c',
						}
						sr.Mac = h.Mac
						sr.ZReport = h.ZReport
						sr.SlipSerial = h.SlipSerial
						sr.DailySlipNo = h.DailySlipNo
						s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
						recordIndex := len(s.v13SlipRecord.Records) - 1
						s.v13SlipRecord.Records[recordIndex].LineC = append(s.v13SlipRecord.Records[recordIndex].LineC, *line2)
					}
				}
			case 'D', 'd':
				{
					line5, h, err := v13.NewLineD(r[1:])
					if err != nil {
						s.errorCode = nexus_errors.ErrWrongNumberOfFields
						return err
					}

					foundLine := false

					for i, t := range s.v13SlipRecord.Records {
						if t.DailySlipNo == h.DailySlipNo &&
							t.SlipSerial == h.SlipSerial &&
							t.ZReport == h.ZReport &&
							t.Mac == h.Mac {
							s.v13SlipRecord.Records[i].LineD = line5
							foundLine = true
						}
					}
					if !foundLine {
						sr := v13.SlipRecord{
							LineD:                     line5,
							IsTransmittedInBackground: line == 'd',
						}
						sr.Mac = h.Mac
						sr.ZReport = h.ZReport
						sr.SlipSerial = h.SlipSerial
						sr.DailySlipNo = h.DailySlipNo
						s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
					}
				}
			case 'E', 'e':
				{
					line6, h, err := v13.NewLineE(r[1:])
					if err != nil {
						s.errorCode = nexus_errors.ErrWrongNumberOfFields
						return err
					}

					foundLine := false

					for i, t := range s.v13SlipRecord.Records {
						if t.DailySlipNo == h.DailySlipNo &&
							t.SlipSerial == h.SlipSerial &&
							t.ZReport == h.ZReport &&
							t.Mac == h.Mac {
							s.v13SlipRecord.Records[i].LineE = line6

							foundLine = true
						}
					}

					if !foundLine {
						sr := v13.SlipRecord{
							LineE:                     line6,
							IsTransmittedInBackground: line == 'e',
						}
						sr.Mac = h.Mac
						sr.ZReport = h.ZReport
						sr.SlipSerial = h.SlipSerial
						sr.DailySlipNo = h.DailySlipNo
						s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
					}

				}
			case 'F', 'f':
				{
					line7, h, err := v13.NewLineF(r[1:])
					if err != nil {
						s.errorCode = nexus_errors.ErrWrongNumberOfFields
						return err
					}
					foundLine := false
					for i, t := range s.v13SlipRecord.Records {
						if t.DailySlipNo == h.DailySlipNo &&
							t.SlipSerial == h.SlipSerial &&
							t.ZReport == h.ZReport &&
							t.Mac == h.Mac {
							s.v13SlipRecord.Records[i].LineF = line7

							foundLine = true
						}
					}
					if !foundLine {
						sr := v13.SlipRecord{
							LineF:                     line7,
							IsTransmittedInBackground: line == 'f',
						}
						sr.Mac = h.Mac
						sr.ZReport = h.ZReport
						sr.SlipSerial = h.SlipSerial
						sr.DailySlipNo = h.DailySlipNo
						s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
					}
				}
			case 'G', 'g':
				{
					line8, h, err := v13.NewLineG(r[1:])
					if err != nil {
						s.errorCode = nexus_errors.ErrWrongNumberOfFields
						return err
					}

					foundLine := false
					for i, t := range s.v13SlipRecord.Records {
						if t.DailySlipNo == h.DailySlipNo &&
							t.SlipSerial == h.SlipSerial &&
							t.ZReport == h.ZReport &&
							t.Mac == h.Mac {
							s.v13SlipRecord.Records[i].LineG = line8

							foundLine = true
						}
					}

					if !foundLine {
						sr := v13.SlipRecord{
							LineG:                     line8,
							IsTransmittedInBackground: line == 'g',
						}
						sr.Mac = h.Mac
						sr.ZReport = h.ZReport
						sr.SlipSerial = h.SlipSerial
						sr.DailySlipNo = h.DailySlipNo
						s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
					}
				}
			case 'H', 'h':
				{
					line9, h, err := v13.NewLineH(r[1:])
					if err != nil {
						s.errorCode = nexus_errors.ErrWrongNumberOfFields
						return err
					}

					foundLine := false
					for i, t := range s.v13SlipRecord.Records {
						if t.DailySlipNo == h.DailySlipNo &&
							t.SlipSerial == h.SlipSerial &&
							t.ZReport == h.ZReport &&
							t.Mac == h.Mac {
							s.v13SlipRecord.Records[i].LineH = line9

							foundLine = true
						}
					}
					if !foundLine {
						sr := v13.SlipRecord{
							LineH:                     line9,
							IsTransmittedInBackground: line == 'h',
						}
						sr.Mac = h.Mac
						sr.ZReport = h.ZReport
						sr.SlipSerial = h.SlipSerial
						sr.DailySlipNo = h.DailySlipNo
						s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
					}
				}
			case 'I', 'i':
				{
					line9, h, err := v13.NewLineI(r[1:])
					if err != nil {
						return err
					}

					foundLine := false
					for i, t := range s.v13SlipRecord.Records {
						if t.DailySlipNo == h.DailySlipNo &&
							t.SlipSerial == h.SlipSerial &&
							t.ZReport == h.ZReport &&
							t.Mac == h.Mac {
							s.v13SlipRecord.Records[i].LineI = line9

							foundLine = true
						}
					}
					if !foundLine {
						sr := v13.SlipRecord{
							LineI:                     line9,
							IsTransmittedInBackground: line == 'i',
						}
						sr.Mac = h.Mac
						sr.ZReport = h.ZReport
						sr.SlipSerial = h.SlipSerial
						sr.DailySlipNo = h.DailySlipNo
						s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
					}
				}
			case 'J', 'j':
				{
					line9, h, err := v13.NewLineJ(r[1:])
					if err != nil {
						return err
					}

					foundLine := false
					for i, t := range s.v13SlipRecord.Records {
						if t.DailySlipNo == h.DailySlipNo &&
							t.SlipSerial == h.SlipSerial &&
							t.ZReport == h.ZReport &&
							t.Mac == h.Mac {
							s.v13SlipRecord.Records[i].LineJ = line9

							foundLine = true
						}
					}
					if !foundLine {
						sr := v13.SlipRecord{
							LineJ:                     line9,
							IsTransmittedInBackground: line == 'j',
						}
						sr.Mac = h.Mac
						sr.ZReport = h.ZReport
						sr.SlipSerial = h.SlipSerial
						sr.DailySlipNo = h.DailySlipNo
						s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
					}
				}
			//case 'K', 'k':
			//	{
			//		line9, h, err := v13.NewLine9(r[1:])
			//		if err != nil {
			//			return err
			//		}
			//
			//		foundLine := false
			//		for i, t := range s.v13SlipRecord.Records {
			//			if t.DailySlipNo == h.DailySlipNo &&
			//				t.SlipSerial == h.SlipSerial &&
			//				t.ZReport == h.ZReport &&
			//				t.Mac == h.Mac {
			//				s.v13SlipRecord.Records[i].L9 = line9
			//
			//				foundLine = true
			//			}
			//		}
			//		if !foundLine {
			//			sr := v13.SlipRecord{
			//				L9: line9,
			//			}
			//			sr.Mac = h.Mac
			//			sr.ZReport = h.ZReport
			//			sr.SlipSerial = h.SlipSerial
			//			sr.DailySlipNo = h.DailySlipNo
			//			sr.EcrSerial = h.EcrSerial
			//			s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
			//		}
			//	}
			//case 'L', 'l':
			//	{
			//		line9, h, err := v13.NewLine9(r[1:])
			//		if err != nil {
			//			return err
			//		}
			//
			//		foundLine := false
			//		for i, t := range s.v13SlipRecord.Records {
			//			if t.DailySlipNo == h.DailySlipNo &&
			//				t.SlipSerial == h.SlipSerial &&
			//				t.ZReport == h.ZReport &&
			//				t.Mac == h.Mac {
			//				s.v13SlipRecord.Records[i].L9 = line9
			//
			//				foundLine = true
			//			}
			//		}
			//		if !foundLine {
			//			sr := v13.SlipRecord{
			//				L9: line9,
			//			}
			//			sr.Mac = h.Mac
			//			sr.ZReport = h.ZReport
			//			sr.SlipSerial = h.SlipSerial
			//			sr.DailySlipNo = h.DailySlipNo
			//			sr.EcrSerial = h.EcrSerial
			//			s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
			//		}
			//	}
			case 'M', 'm':
				{
					line9, h, err := v13.NewLineM(r[1:])
					if err != nil {
						s.errorCode = nexus_errors.ErrWrongNumberOfFields
						return err
					}

					foundLine := false
					for i, t := range s.v13SlipRecord.Records {
						if t.ZReport == h.ZReport &&
							t.Mac == h.Mac &&
							t.LineM != nil &&
							t.LineM.RecordNo == line9.RecordNo {
							s.v13SlipRecord.Records[i].LineM = line9

							foundLine = true
						}
					}
					if !foundLine {
						sr := v13.SlipRecord{
							LineM:                     line9,
							IsTransmittedInBackground: line == 'm',
						}
						sr.Mac = h.Mac
						sr.ZReport = h.ZReport
						sr.SlipSerial = h.SlipSerial
						sr.DailySlipNo = h.DailySlipNo
						s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
					}
				}
			case 'N', 'n':
				{
					//TODO versioni aktual i protokollit nuk ka header per kete rresht
					//line9, h, err := v13.NewLineN(r[1:])
					//if err != nil {
					//	return err
					//}
					//
					//foundLine := false
					//for i, t := range s.v13SlipRecord.Records {
					//	if t.DailySlipNo == h.DailySlipNo &&
					//		t.SlipSerial == h.SlipSerial &&
					//		t.ZReport == h.ZReport &&
					//		t.Mac == h.Mac {
					//		s.v13SlipRecord.Records[i].LineN = line9
					//
					//		foundLine = true
					//	}
					//}
					//if !foundLine {
					//	sr := v13.SlipRecord{
					//		LineN:                     line9,
					//		IsTransmittedInBackground: line == 'n',
					//	}
					//	sr.Mac = h.Mac
					//	sr.ZReport = h.ZReport
					//	sr.SlipSerial = h.SlipSerial
					//	sr.DailySlipNo = h.DailySlipNo
					//	s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
					//}
				}
			case 'S', 's':
				{
					level.Info(s.l).Log("LineS", hex.EncodeToString(r[1:]))
					//line9, h, err := v13.NewLine9(r[1:])
					//if err != nil {
					//	return err
					//}
					//
					//foundLine := false
					//for i, t := range s.v13SlipRecord.Records {
					//	if t.DailySlipNo == h.DailySlipNo &&
					//		t.SlipSerial == h.SlipSerial &&
					//		t.ZReport == h.ZReport &&
					//		t.Mac == h.Mac {
					//		s.v13SlipRecord.Records[i].L9 = line9
					//
					//		foundLine = true
					//	}
					//}
					//if !foundLine {
					//	sr := v13.SlipRecord{
					//		L9: line9,
					//	}
					//	sr.Mac = h.Mac
					//	sr.ZReport = h.ZReport
					//	sr.SlipSerial = h.SlipSerial
					//	sr.DailySlipNo = h.DailySlipNo
					//	sr.EcrSerial = h.EcrSerial
					//	s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
					//}
				}
			case 'T', 't':
				{
					line9, h, err := v13.NewLineT(r[1:])
					if err != nil {
						return err
					}

					foundLine := false
					for i, t := range s.v13SlipRecord.Records {
						if t.DailySlipNo == h.DailySlipNo &&
							t.SlipSerial == h.SlipSerial &&
							t.ZReport == h.ZReport &&
							t.Mac == h.Mac {
							s.v13SlipRecord.Records[i].LineT = line9

							foundLine = true
						}
					}
					if !foundLine {
						sr := v13.SlipRecord{
							LineT: line9,
						}
						sr.Mac = h.Mac
						sr.ZReport = h.ZReport
						sr.SlipSerial = h.SlipSerial
						sr.DailySlipNo = h.DailySlipNo
						s.v13SlipRecord.Records = append(s.v13SlipRecord.Records, sr)
					}
				}
			default:
				{
					//TODO duhet
					//s.errorCode = nexus_errors.ErrUnknownSlipLine
					//return fmt.Errorf("unknown slip line %X", line)
				}
			}
		}
	}
	return nil
}
