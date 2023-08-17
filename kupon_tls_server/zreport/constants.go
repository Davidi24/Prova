package zreport

const (
	ZReportMaxMessageLength = 4147

	ZReportHeaderLength = 54

	ZReportMessageIdentifier = 87 // 'W'

	ZReportIdentifierLength = 1
	ZReportIdentifierOffset = 0
	ZReportIdentifierLast   = ZReportIdentifierLength +
		ZReportIdentifierOffset

	ZReportProtocolLength = 2
	ZReportProtocolOffset = 1
	ZReportProtocolLast   = ZReportProtocolOffset + ZReportProtocolLength

	ZReportLengthFieldLength = 2
	ZReportLengthFieldOffset = 3
	ZReportLengthFieldLast   = ZReportLengthFieldLength + ZReportLengthFieldOffset

	ZReportTypeLength = 1
	ZReportTypeOffset = 5
	ZReportTypeLast   = ZReportTypeLength + ZReportTypeOffset

	ZReportEcrSerialLength = 10
	ZReportEcrSerialOffset = 6
	ZReportEcrSerialLast   = ZReportEcrSerialLength + ZReportEcrSerialOffset

	ZReportEcrFileNameLength = 38
	ZReportEcrFileNameOffset = 16
	ZReportEcrFileNameLast   = ZReportEcrFileNameLength + ZReportEcrFileNameOffset

	ZReportEcrFileContentOffset = 54

	ZReportCheckSumLength = 2

	ZReportProtocolACK = "A0000"
)
