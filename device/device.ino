#include "TelenorNBIoT.h"

#include <pb_encode.h>
#include "message.pb.h"

// Magic for selecting serial port
#ifdef SERIAL_PORT_HARDWARE_OPEN
/*
 * For Arduino boards with a hardware serial port separate from USB serial.
 * This is usually mapped to Serial1. Check which pins are used for Serial1 on
 * the board you're using.
 */
#define ublox SERIAL_PORT_HARDWARE_OPEN
#else
/*
 * For Arduino boards with only one hardware serial port (like Arduino UNO). It
 * is mapped to USB, so we use SoftwareSerial on pin 10 and 11 instead.
 */
#include <SoftwareSerial.h>
SoftwareSerial ublox(10, 11);
#endif

// Configure mobile country code, mobile network code and access point name
// See https://www.mcc-mnc.com/ for country and network codes
// Access Point Namme: mda.ee (Telenor NB-IoT Developer Platform)
// Mobile Country Code: 242 (Norway)
// Mobile Network Operator: 01 (Telenor)
TelenorNBIoT nbiot("mda.ee", 242, 01);

IPAddress remoteIP(172, 16, 15, 14);
int REMOTE_PORT = 1234;

void setup() {
	Serial.begin(9600);
	while (!Serial) {}

	ublox.begin(9600);
	nbiot.begin(ublox);

	Serial.print(F("waiting for connection..."));
	while (!nbiot.isConnected()) {
		Serial.print(F("."));
		delay(1000);
	}
	Serial.println(F("connected"));

	Serial.print(F("creating socket..."));
	while (!nbiot.createSocket()) {
		Serial.print(F("."));
		delay(1000);
	}
	Serial.println(F("created socket"));
}

uint32_t sequence = 1;

void loop() {
    nbiot_e2e_Message msg = {
        .which_message = nbiot_e2e_Message_ping_message_tag,
        .message = {
            .ping_message = {
            	.sequence = sequence,
            },
        },
    };

	uint8_t msg_buffer[nbiot_e2e_Message_size] = { 0 };
	pb_ostream_t stream = pb_ostream_from_buffer(msg_buffer, sizeof(msg_buffer));
	if (!pb_encode(&stream, nbiot_e2e_Message_fields, &msg)) {
		Serial.println(sprintf("pb_encode error: %s\n", stream.errmsg));
		goto end;
	}

	if (nbiot.sendBytes(remoteIP, REMOTE_PORT, (const char*)msg_buffer, stream.bytes_written)) {
		Serial.println(F("sent message"));
		++sequence;
	} else {
		Serial.println(F("failed to send"));
		goto end;
	}

end:
	delay(15000);
}
