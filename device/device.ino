#include "TelenorNBIoT.h"

#include <pb_encode.h>
#include "message.pb.h"

// https://stackoverflow.com/a/5256500/44643
#define CAT_NX(A, B) A ## B
#define CAT(A, B) CAT_NX(A, B)

#ifndef NBIOT_LIB_HASH
#define NBIOT_LIB_HASH
#warning Missing nbiot library commit hash
#endif

#ifndef E2E_HASH
#define E2E_HASH
#warning Missing e2e commit hash
#endif

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

uint32_t nbiot_lib_hash = CAT(0x0, NBIOT_LIB_HASH);
uint32_t e2e_hash = CAT(0x0, E2E_HASH);


void setup() {
	Serial.begin(9600);
	while (!Serial) {}

	Serial.print("ArduinoNBIoT git commit: ");
	Serial.println(nbiot_lib_hash, HEX);
	Serial.print("nbiot-e2e git commit: ");
	Serial.println(e2e_hash, HEX);

	ublox.begin(9600);
	nbiot.begin(ublox);

	Serial.println(F("starting nbiot e2e test"));
	Serial.print(F("waiting for connection"));
	while (!nbiot.isConnected()) {
		printSignalStrength();
		delay(1000);
	}
	Serial.println(F("connected"));

	Serial.print(F("creating socket"));
	while (!nbiot.createSocket()) {
		delay(1000);
	}
	Serial.println(F("created socket"));
}

uint32_t sequence = 1;

void loop() {
	printSignalStrength();

    nbiot_e2e_Message msg = {
        which_message: nbiot_e2e_Message_ping_message_tag,
        message: {
            ping_message: {
            	sequence:       sequence,
				rssi:           99.0,
				nbiot_lib_hash: nbiot_lib_hash,
				e2e_hash:       e2e_hash,
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

void printSignalStrength() {
	int rssi = nbiot.rssi();
	Serial.print(F("signal strength: "));
	if (rssi == 99) {
		Serial.println(F("unknown"));
	} else {
		Serial.print(rssi);
		Serial.println(F(" dBm"));
	}
}
