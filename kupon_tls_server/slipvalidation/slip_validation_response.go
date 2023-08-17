package slipvalidation

import (
	"bytes"
	"encoding/binary"
	"nexusws/pkg/checksum"
)

func NewResponse(protocolVersion string, url string) *response {
	return &response{
		MessageIdentifier: 72, //H
		ProtocolVersion:   protocolVersion,
		QrCodeType:        QrCodeTypeUrl,
		QrCodeUrlLength:   uint16(len(url)),
		UrlQrcode:         url,
	}
}

const (
	QrCodeTypeUrl    = 65 // A
	QrCodeTypeBitmap = 65 // B
)

type response struct {
	MessageIdentifier uint8
	ProtocolVersion   string
	QrCodeUrlLength   uint16
	QrCodeType        uint8
	UrlQrcode         string
	Checksum          string
}

func (r response) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.LittleEndian, r.MessageIdentifier)
	if err != nil {
		return nil, err
	}

	_, err = buf.Write([]byte(r.ProtocolVersion))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, r.QrCodeUrlLength)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.LittleEndian, r.QrCodeType)
	if err != nil {
		return nil, err
	}

	_, err = buf.Write([]byte(r.UrlQrcode))
	if err != nil {
		return nil, err
	}

	cs := checksum.CalcXorChecksum(buf.Bytes())
	_, err = buf.Write([]byte(cs))
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
