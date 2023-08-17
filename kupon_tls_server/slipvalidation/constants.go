package slipvalidation

const (
	SlipValidationMaxMessageLength = 66
	SlipValidationRecordBodyOffset = 32
	SlipValidationHeaderLength     = 32

	SlipValidationIdentifierLength = 1
	SlipValidationIdentifierOffset = 0
	SlipValidationIdentifierLast   = SlipValidationIdentifierLength +
		SlipValidationIdentifierOffset

	SlipValidationProtocolLength = 2
	SlipValidationProtocolOffset = 1
	SlipValidationProtocolLast   = SlipValidationProtocolOffset + SlipValidationProtocolLength

	SlipValidationEcrSerialLength  = 10
	SlipValidationEcrSerialdOffset = 3
	SlipValidationEcrSerialLast    = SlipValidationEcrSerialLength + SlipValidationEcrSerialdOffset

	SlipValidationMacLength = 3
	SlipValidationMacOffset = 13
	SlipValidationMacLast   = SlipValidationMacLength + SlipValidationMacOffset

	SlipValidationRapZLength = 4
	SlipValidationRapZOffset = 16
	SlipValidationRapZLast   = SlipValidationRapZLength + SlipValidationRapZOffset

	SlipValidationDailySlipNoLength = 4
	SlipValidationDailySlipNoOffset = 20
	SlipValidationDailySlipNoLast   = SlipValidationDailySlipNoLength + SlipValidationDailySlipNoOffset

	SlipValidationSerialLength = 8
	SlipValidationSerialOffset = 24
	SlipValidationSerialLast   = SlipValidationSerialLength + SlipValidationSerialOffset

	SlipValidationMD5Length = 32
	SlipValidationMD5Offset = 32
	SlipValidationMD5Last   = SlipValidationMD5Length + SlipValidationMD5Offset

	SlipValidationCheckSumLength = 2
	SlipValidationCheckSumOffset = 64
	SlipValidationCheckSumLast   = SlipValidationCheckSumLength + SlipValidationCheckSumOffset

	SlipValidationProtocolACK = "A0000"
)
