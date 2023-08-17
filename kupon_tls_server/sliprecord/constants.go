package sliprecord

const (
	SlipMaxMessageLength = 2048
	// SlipRecordBodyOffset   = 24

	SlipRecordMessageIdentifier     = 71 //'G'
	SlipValidationMessageIdentifier = 69 // 'E'

	SlipRecordIdentifierLength = 1
	SlipRecordIdentifierOffset = 0
	SlipRecordIdentifierLast   = SlipRecordIdentifierLength +
		SlipRecordIdentifierOffset

	SlipRecordProtocolLength = 2
	SlipRecordProtocolOffset = 1
	SlipRecordProtocolLast   = SlipRecordProtocolOffset + SlipRecordProtocolLength

	SlipRecordLengthFieldLength = 2
	SlipRecordLengthFieldOffset = 3
	SlipRecordLengthFieldLast   = SlipRecordLengthFieldLength + SlipRecordLengthFieldOffset

	SlipRecordTypeLength = 1
	SlipRecordTypeOffset = 5
	SlipRecordTypeLast   = SlipRecordTypeLength + SlipRecordTypeOffset

	SlipRecordEcrSerialLength = 10
	SlipRecordEcrSerialOffset = 6
	SlipRecordEcrSerialLast   = SlipRecordEcrSerialLength + SlipRecordEcrSerialOffset

	SlipRecordFirstRecordLength = 4
	SlipRecordFirstRecordOffset = 16
	SlipRecordFirstRecordLast   = SlipRecordFirstRecordLength + SlipRecordFirstRecordOffset

	SlipRecordLastLength = 4
	SlipRecordLastOffset = 20
	SlipRecordLastLast   = SlipRecordLastLength + SlipRecordLastOffset

	SlipRecordV13HeaderLength = SlipRecordIdentifierLength +
		SlipRecordProtocolLength +
		SlipRecordLengthFieldLength +
		SlipRecordTypeLength +
		SlipRecordEcrSerialLength

	SlipRecordCheckSumLength = 2

	SlipRecordProtocolACK = "A0000"
)
